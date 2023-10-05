package sync

import (
	"context"
	"fmt"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// processPendingBlobs listens for state changes and handles pending blobs.
func (s *Service) processPendingBlobs() {
	eventFeed := make(chan *feed.Event, 1)
	sub := s.stateNotifier.StateFeed().Subscribe(eventFeed)
	defer sub.Unsubscribe()

	slotTicker := slots.NewSlotTicker(s.cfg.chain.GenesisTime(), params.BeaconConfig().SecondsPerSlot)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-sub.Err():
			return
		case e := <-eventFeed:
			s.handleEvent(s.ctx, e)
		case slot := <-slotTicker.C():
			// Prune sidecars older than previous slot.
			if slot > 0 {
				slot--
			}
			s.pruneOldSidecars(slot)
		}
	}
}

// handleEvent processes incoming events.
func (s *Service) handleEvent(ctx context.Context, e *feed.Event) {
	if e.Type == statefeed.BlockProcessed {
		s.handleNewBlockEvent(ctx, e)
	}
}

// handleNewBlockEvent handles blobs when a parent block is processed.
func (s *Service) handleNewBlockEvent(ctx context.Context, e *feed.Event) {
	data, ok := e.Data.(*statefeed.BlockProcessedData)
	if !ok {
		return
	}
	s.processBlobsFromSidecars(ctx, data.SignedBlock.Block().ParentRoot())
}

// processBlobsFromSidecars processes blobs for a given parent root.
func (s *Service) processBlobsFromSidecars(ctx context.Context, parentRoot [32]byte) {
	blobs := s.pendingBlobSidecars.pop(parentRoot)
	for _, blob := range blobs {
		if err := s.validateAndReceiveBlob(ctx, blob); err != nil {
			log.WithError(err).Error("Failed to validate blob in pending queue")
		}
	}
}

// validateAndReceiveBlob validates and receives a blob if it's valid.
func (s *Service) validateAndReceiveBlob(ctx context.Context, blob *eth.SignedBlobSidecar) error {
	result, err := s.validateBlobPostSeenParent(ctx, blob)
	if err != nil {
		return err
	}
	if result != pubsub.ValidationAccept {
		return fmt.Errorf("unexpected pubsub result: %d", result)
	}
	return s.cfg.chain.ReceiveBlob(ctx, blob.Message)
}

// pruneOldSidecars removes sidecars older than a given slot.
func (s *Service) pruneOldSidecars(slot primitives.Slot) {
	s.pendingBlobSidecars.pruneOlderThanSlot(slot)
}

// pendingBlobSidecars holds pending blob sidecars.
type pendingBlobSidecars struct {
	sync.RWMutex
	blobSidecars map[[32]byte][]*eth.SignedBlobSidecar
}

// newPendingBlobSidecars initializes a new pendingBlobSidecars instance.
func newPendingBlobSidecars() *pendingBlobSidecars {
	return &pendingBlobSidecars{blobSidecars: make(map[[32]byte][]*eth.SignedBlobSidecar)}
}

// add adds a new blob sidecar to the pending queue.
func (p *pendingBlobSidecars) add(blob *eth.SignedBlobSidecar) {
	p.Lock()
	defer p.Unlock()

	parentRoot := bytesutil.ToBytes32(blob.Message.BlockParentRoot)
	p.blobSidecars[parentRoot] = append(p.blobSidecars[parentRoot], blob)
}

// pop removes and returns all blob sidecars for a given parent root.
func (p *pendingBlobSidecars) pop(parentRoot [32]byte) []*eth.SignedBlobSidecar {
	p.Lock()
	defer p.Unlock()

	blobs, exists := p.blobSidecars[parentRoot]
	if exists {
		delete(p.blobSidecars, parentRoot)
	}
	return blobs
}

// pruneOlderThanSlot removes all blob sidecars older than a given slot.
func (p *pendingBlobSidecars) pruneOlderThanSlot(slot primitives.Slot) {
	p.Lock()
	defer p.Unlock()

	for root, sidecars := range p.blobSidecars {
		if allOlderThanSlot(sidecars, slot) {
			delete(p.blobSidecars, root)
		}
	}
}

// allOlderThanSlot checks if all blob sidecars are older than a given slot.
func allOlderThanSlot(sidecars []*eth.SignedBlobSidecar, slot primitives.Slot) bool {
	for _, sidecar := range sidecars {
		if sidecar.Message.Slot > slot {
			return false
		}
	}
	return true
}

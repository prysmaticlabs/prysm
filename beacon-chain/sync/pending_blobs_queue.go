package sync

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// processPendingBlobs listens to state changes and handles pending blobs.
func (s *Service) processPendingBlobs() {
	ctx := context.Background()
	eventFeed := make(chan *feed.Event, 1)
	sub := s.stateNotifier.StateFeed().Subscribe(eventFeed)
	defer sub.Unsubscribe()

	for {
		select {
		case <-s.ctx.Done(): // exit if context is done
			return
		case <-sub.Err(): // exit on subscription error
			return
		case e := <-eventFeed:
			s.handleEvent(ctx, e)
		}
	}
}

// handleEvent processes an incoming event, dispatching it to the appropriate handler function.
func (s *Service) handleEvent(ctx context.Context, e *feed.Event) {
	switch e.Type {
	case statefeed.BlockProcessed:
		s.handleNewBlockEvent(ctx, e)
	case statefeed.FinalizedCheckpoint:
		s.pruneBlobsOnFinalization(e)
	}
}

// handleNewBlockEvent processes blobs when a parent block is processed.
func (s *Service) handleNewBlockEvent(ctx context.Context, e *feed.Event) {
	data, ok := e.Data.(*statefeed.BlockProcessedData)
	if !ok {
		// Ignore if data is of the wrong type
		return
	}
	parentRoot := data.SignedBlock.Block().ParentRoot()

	s.pendingBlobSidecars.Lock()
	blobs, exists := s.pendingBlobSidecars.blobSidecars[parentRoot]
	if exists {
		delete(s.pendingBlobSidecars.blobSidecars, parentRoot)
	}
	s.pendingBlobSidecars.Unlock()

	for _, blob := range blobs {
		if err := s.validateAndReceiveBlob(ctx, blob); err != nil {
			log.WithError(err).Error("Blob validation failed in pending queue")
		}
	}
}

// validateAndReceiveBlob validates a blob and receives it if valid.
func (s *Service) validateAndReceiveBlob(ctx context.Context, blob *eth.SignedBlobSidecar) error {
	pubsubResult, err := s.validateBlobPostSeenParent(ctx, blob)
	if err != nil {
		return err
	}
	if pubsubResult != pubsub.ValidationAccept {
		return fmt.Errorf("pubsubResult is not ValidationAccept, got %d", pubsubResult)
	}
	return s.cfg.chain.ReceiveBlob(ctx, blob.Message)
}

// pruneBlobsOnFinalization removes staled blobs older than the finalized checkpoint.
func (s *Service) pruneBlobsOnFinalization(e *feed.Event) {
	finalizedCheckpoint, ok := e.Data.(*ethpb.EventFinalizedCheckpoint)
	if !ok {
		// Ignore if data is of the wrong type
		return
	}
	finalizedSlot, err := slots.EpochStart(finalizedCheckpoint.Epoch)
	if err != nil {
		log.WithError(err).Error("Could not get finalized slot")
		return
	}

	// Remove blobs older than finalized checkpoint
	s.pendingBlobSidecars.Lock()
	for parentRoot, sidecars := range s.pendingBlobSidecars.blobSidecars {
		if s.isOlderThanFinalizedSlot(sidecars, finalizedSlot) {
			delete(s.pendingBlobSidecars.blobSidecars, parentRoot)
		}
	}
	s.pendingBlobSidecars.Unlock()
}

// isOlderThanFinalizedSlot checks if all blobs are older than the finalized slot.
func (s *Service) isOlderThanFinalizedSlot(sidecars []*eth.SignedBlobSidecar, finalizedSlot primitives.Slot) bool {
	for _, sidecar := range sidecars {
		if sidecar.Message.Slot > finalizedSlot {
			return false
		}
	}
	return true
}

// pendingBlobSidecars contains pending blob sidecars and a mutex for concurrent access.
type pendingBlobSidecars struct {
	sync.RWMutex
	blobSidecars map[[32]byte][]*eth.SignedBlobSidecar
}

// newPendingBlobSidecars creates a new pendingBlobSidecars instance.
func newPendingBlobSidecars() *pendingBlobSidecars {
	return &pendingBlobSidecars{
		blobSidecars: make(map[[32]byte][]*eth.SignedBlobSidecar),
	}
}

// add adds a new blob sidecar to the pending list.
func (p *pendingBlobSidecars) add(blob *eth.SignedBlobSidecar) {
	p.Lock()
	defer p.Unlock()

	parentRoot := bytesutil.ToBytes32(blob.Message.BlockParentRoot)
	for _, existingBlob := range p.blobSidecars[parentRoot] {
		// Check for duplicates
		if existingBlob.Message.Index == blob.Message.Index && bytes.Equal(existingBlob.Message.BlockRoot, blob.Message.BlockRoot) {
			return
		}
	}

	p.blobSidecars[parentRoot] = append(p.blobSidecars[parentRoot], blob)
}

package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// processPendingBlobs listens for state changes and handles pending blobs.
func (s *Service) processPendingBlobs() {
	s.waitForChainStart()

	eventFeed := make(chan *feed.Event, 1)
	sub := s.cfg.stateNotifier.StateFeed().Subscribe(eventFeed)
	defer sub.Unsubscribe()

	// Initialize the cleanup ticker at 11s mark. The node is less busy that time.
	cleanupTicker := slots.NewSlotTickerWithIntervals(s.cfg.clock.GenesisTime(), []time.Duration{time.Duration(params.BeaconConfig().SecondsPerSlot-1) * time.Second} /* 11s */)

	for {
		select {
		case <-s.ctx.Done():
			log.Debug("Stopping pending blobs queue")
			return
		case <-sub.Err():
			log.Debugf("Stopping pending blobs queue due to state feed error: %v", sub.Err())
			return
		case e := <-eventFeed:
			s.handleEvent(s.ctx, e)
		case <-cleanupTicker.C():
			if s.pendingBlobSidecars == nil {
				return
			}
			s.pendingBlobSidecars.cleanup()
		}
	}
}

// handleEvent processes incoming events.
func (s *Service) handleEvent(ctx context.Context, e *feed.Event) {
	if e.Type == statefeed.BlockProcessed {
		s.handleNewBlockEvent(ctx, e)
	}
}

// handleNewBlockEvent processes blobs when a parent block is processed.
func (s *Service) handleNewBlockEvent(ctx context.Context, e *feed.Event) {
	data, ok := e.Data.(*statefeed.BlockProcessedData)
	if !ok {
		return
	}
	if data == nil || data.SignedBlock.IsNil() {
		return
	}
	s.processBlobsFromSidecars(ctx, data.SignedBlock.Block().ParentRoot())
}

// processBlobsFromSidecars processes blobs for a given parent root.
func (s *Service) processBlobsFromSidecars(ctx context.Context, parentRoot [32]byte) {
	blobs := s.pendingBlobSidecars.pop(parentRoot)
	for _, blob := range blobs {
		if err := s.receiveBlob(ctx, blob); err != nil {
			log.WithError(err).Error("Failed to validate blob in pending queue")
		}
	}
}

// receiveBlob validates and processes a blob.
func (s *Service) receiveBlob(ctx context.Context, blob *eth.SignedBlobSidecar) error {
	result, err := s.validateBlobPostSeenParent(ctx, blob)
	if err != nil {
		return err
	}
	if result != pubsub.ValidationAccept {
		return fmt.Errorf("unexpected pubsub result: %d", result)
	}
	return s.cfg.chain.ReceiveBlob(ctx, blob.Message)
}

// blobWithExpiration holds blobs with an expiration time.
type blobWithExpiration struct {
	blob      []*eth.SignedBlobSidecar
	expiresAt time.Time
}

// pendingBlobSidecars holds pending blobs with expiration.
type pendingBlobSidecars struct {
	sync.RWMutex
	blobSidecars map[[32]byte]*blobWithExpiration // Key is the block root.
}

// newPendingBlobSidecars initializes a new cache of pending blobs.
func newPendingBlobSidecars() *pendingBlobSidecars {
	return &pendingBlobSidecars{
		blobSidecars: make(map[[32]byte]*blobWithExpiration),
	}
}

// add adds a new blob to the cache.
func (p *pendingBlobSidecars) add(blob *eth.SignedBlobSidecar) {
	p.Lock()
	defer p.Unlock()
	parentRoot := bytesutil.ToBytes32(blob.Message.BlockParentRoot)
	expirationTime := time.Now().Add(time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)

	if existing, exists := p.blobSidecars[parentRoot]; exists {
		for _, sidecar := range existing.blob {
			if sidecar.Message.Index == blob.Message.Index {
				return // Ignore duplicate blob index
			}
		}
		existing.blob = append(existing.blob, blob)
	} else {
		p.blobSidecars[parentRoot] = &blobWithExpiration{
			blob:      []*eth.SignedBlobSidecar{blob},
			expiresAt: expirationTime,
		}
	}
}

// pop removes and returns blobs for a given parent root.
func (p *pendingBlobSidecars) pop(parentRoot [32]byte) []*eth.SignedBlobSidecar {
	p.Lock()
	defer p.Unlock()
	blobs, exists := p.blobSidecars[parentRoot]
	if exists {
		delete(p.blobSidecars, parentRoot)
	}
	if blobs != nil {
		return blobs.blob
	}
	return nil // Return nil if blobs does not exist
}

// cleanup removes expired blobs from the cache.
func (p *pendingBlobSidecars) cleanup() {
	p.Lock()
	defer p.Unlock()
	now := time.Now()
	for root, blobInfo := range p.blobSidecars {
		if blobInfo.expiresAt.Before(now) {
			delete(p.blobSidecars, root)
		}
	}
}

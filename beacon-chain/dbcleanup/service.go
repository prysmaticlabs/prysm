// Package dbcleanup defines the life cycle and logic of beacon DB cleanup routine.
package dbcleanup

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "dbcleaner")

type chainService interface {
	CanonicalStateFeed() *event.Feed
}

// CleanupService represents a service that handles routine task for cleaning up
// beacon DB so our DB won't grow infinitely.
// Currently it only cleans up block vote cache. In future, it could add more tasks
// such as cleaning up historical beacon states.
type CleanupService struct {
	ctx                context.Context
	cancel             context.CancelFunc
	beaconDB           *db.BeaconDB
	chainService       chainService
	canonicalStateChan chan *pb.BeaconState
}

// Config defines the needed fields for creating a new cleanup service.
type Config struct {
	SubscriptionBuf int
	BeaconDB        *db.BeaconDB
	ChainService    chainService
}

// NewCleanupService creates a new cleanup service instance.
func NewCleanupService(ctx context.Context, cfg *Config) *CleanupService {
	ctx, cancel := context.WithCancel(ctx)
	return &CleanupService{
		ctx:                ctx,
		cancel:             cancel,
		beaconDB:           cfg.BeaconDB,
		chainService:       cfg.ChainService,
		canonicalStateChan: make(chan *pb.BeaconState, cfg.SubscriptionBuf),
	}
}

// Start a cleanup service.
func (d *CleanupService) Start() {
	log.Info("Starting service")
	go d.cleanDB()
}

// Stop a cleanup service.
func (d *CleanupService) Stop() error {
	defer d.cancel()

	log.Info("Stopping service")
	return nil
}

// Status always returns nil.
// TODO(1203): Add service health checks.
func (d *CleanupService) Status() error {
	return nil
}

func (d *CleanupService) cleanDB() {
	cStateSub := d.chainService.CanonicalStateFeed().Subscribe(d.canonicalStateChan)
	defer cStateSub.Unsubscribe()

	for {
		select {
		case <-d.ctx.Done():
			log.Debug("Cleanup service context closed, exiting goroutine")
			return
		case cState := <-d.canonicalStateChan:
			if err := d.cleanBlockVoteCache(cState.FinalizedEpoch); err != nil {
				log.Errorf("Failed to clean block vote cache: %v", err)
			}
		}
	}
}

func (d *CleanupService) cleanBlockVoteCache(latestFinalizedSlot uint64) error {
	var lastCleanedFinalizedSlot uint64
	var err error

	lastCleanedFinalizedSlot, err = d.beaconDB.CleanedFinalizedSlot()
	if err != nil {
		return fmt.Errorf("failed to read cleaned finalized slot from DB: %v", err)
	}

	log.Infof("Finalized slot: latest: %d, last cleaned: %d, %d blocks' vote cache will be cleaned",
		latestFinalizedSlot, lastCleanedFinalizedSlot, latestFinalizedSlot-lastCleanedFinalizedSlot)

	var blockHashes [][32]byte
	for slot := lastCleanedFinalizedSlot + 1; slot <= latestFinalizedSlot; slot++ {
		var block *pb.BeaconBlock
		block, err = d.beaconDB.BlockBySlot(slot)
		if err != nil {
			return fmt.Errorf("failed to read block at slot %d: %v", slot, err)
		}
		if block != nil {
			var blockHash [32]byte
			blockHash, err = hashutil.HashBeaconBlock(block)
			if err != nil {
				return fmt.Errorf("failed to get hash of block: %v", err)
			}
			blockHashes = append(blockHashes, blockHash)
		}
	}
	if err = d.beaconDB.DeleteBlockVoteCache(blockHashes); err != nil {
		return fmt.Errorf("failed to delete block vote cache: %v", err)
	}

	if err = d.beaconDB.SaveCleanedFinalizedSlot(latestFinalizedSlot); err != nil {
		return fmt.Errorf("failed to update cleaned finalized slot: %v", err)
	}
	return nil
}

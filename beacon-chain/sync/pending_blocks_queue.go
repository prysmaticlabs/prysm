package sync

import (
	"context"
	"sort"
	"time"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var processPendingBlocksPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot/2) * time.Second

// processes pending blocks queue on every processPendingBlocksPeriod
func (r *RegularSync) processPendingBlocksQueue(ctx context.Context) {
	ticker := time.NewTicker(processPendingBlocksPeriod)
	for {
		select {
		case <-ticker.C:
			r.processPendingBlocks(ctx)
		case <-r.ctx.Done():
			log.Debug("p2p context is closed, exiting routine")
			break

		}
	}
}

// processes the block tree inside the queue
func (r *RegularSync) processPendingBlocks(ctx context.Context) error {
	// Construct a sorted list of slots from outstanding pending blocks
	r.slotToPendingBlocksLock.Lock()
	r.seenPendingBlocksLock.Lock()
	defer r.slotToPendingBlocksLock.Unlock()
	defer r.seenPendingBlocksLock.Unlock()
	slots := make([]int, 0, len(r.slotToPendingBlocks))
	for s := range r.slotToPendingBlocks {
		slots = append(slots, int(s))
	}
	sort.Ints(slots)
	// For every pending block, process block if parent exists
	log.Info("Processing pending blocks ", slots)
	for _, s := range slots {
		b := r.slotToPendingBlocks[uint64(s)]
		if !r.db.HasBlock(ctx, bytesutil.ToBytes32(b.ParentRoot)) {
			continue
		}
		if err := r.chain.ReceiveBlockNoPubsub(ctx, b); err != nil {
			log.Errorf("Could not process block from slot %d: %v", b.Slot, err)
		}
		bRoot, err := ssz.SigningRoot(b)
		if err != nil {
			return err
		}
		delete(r.slotToPendingBlocks, uint64(s))
		delete(r.seenPendingBlocks, bRoot)
		log.Infof("Processed ancestor block %d and cleared pending block cache", s)
	}
	return nil
}

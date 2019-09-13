package sync

import (
	"context"
	"sort"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func (r *RegularSync) beaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	block := msg.(*ethpb.BeaconBlock)

	headState := r.chain.HeadState()

	// Ignore block older than last finalized checkpoint.
	if block.Slot < helpers.StartSlot(headState.FinalizedCheckpoint.Epoch) {
		log.Debugf("Received a block that's older than finalized checkpoint, %d < %d",
			block.Slot, helpers.StartSlot(headState.FinalizedCheckpoint.Epoch))
		return nil
	}

	// Ignore block already in pending blocks cache
	if _, ok := r.pendingBlocks[block.Slot]; ok {
		return nil
	}

	// Handle block when the parent is unknown
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(block.ParentRoot)) {
		r.pendingBlocksLock.Lock()
		defer r.pendingBlocksLock.Unlock()
		r.pendingBlocks[block.Slot] = block

		// TODO(3147): Request parent block from peers
		log.Warnf("Received a block from slot %d which we do not have the parent block in the database. "+
			"Requesting parent.", block.Slot)
		return nil
	}

	// Attempt to process blocks from the saved pending blocks
	ticker := time.NewTicker(time.Duration(params.BeaconConfig().SecondsPerSlot / 2))
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				r.processPendingBlocks(ctx)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	return r.chain.ReceiveBlockNoPubsub(ctx, block)
}

func (r *RegularSync) processPendingBlocks(ctx context.Context) {
	// Construct a sorted list of slots from outstanding pending blocks
	r.pendingBlocksLock.Lock()
	defer r.pendingBlocksLock.Unlock()
	slots := make([]int, 0, len(r.pendingBlocks))
	for s := range r.pendingBlocks {
		slots = append(slots, int(s))
	}
	sort.Ints(slots)
	// For every pending block, process block if parent exists
	for _, s := range slots {
		b := r.pendingBlocks[uint64(s)]
		if !r.db.HasBlock(ctx, bytesutil.ToBytes32(b.ParentRoot)) {
			continue
		}
		if err := r.chain.ReceiveBlockNoPubsub(ctx, b); err != nil {
			log.Errorf("Could not process block from slot %d: %v", b.Slot, err)
		}
		delete(r.pendingBlocks, uint64(s))
	}
}

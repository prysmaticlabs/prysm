package sync

import (
	"context"
	"sort"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
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
		r.seenPendingBlocksLock.Lock()
		defer r.pendingBlocksLock.Unlock()
		defer r.seenPendingBlocksLock.Unlock()
		blockRoot, err := ssz.SigningRoot(block)
		if err != nil {
			log.Errorf("Could not sign root block: %v", err)
			return nil
		}
		r.pendingBlocks[block.Slot] = block
		r.seenPendingBlocks[blockRoot] = true

		if !r.seenPendingBlocks[bytesutil.ToBytes32(block.ParentRoot)] {
			log.Warnf("Received a block from slot %d which we do not have the parent block in the database. "+
				"Requesting parent.", block.Slot)
		}

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

func (r *RegularSync) processPendingBlocks(ctx context.Context) error {
	// Construct a sorted list of slots from outstanding pending blocks
	r.pendingBlocksLock.Lock()
	r.seenPendingBlocksLock.Lock()
	defer r.pendingBlocksLock.Unlock()
	defer r.seenPendingBlocksLock.Unlock()
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
		bRoot, err := ssz.SigningRoot(b)
		if err != nil {
			return err
		}
		delete(r.pendingBlocks, uint64(s))
		delete(r.seenPendingBlocks, bRoot)
	}
	return nil
}

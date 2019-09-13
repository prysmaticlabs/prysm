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

var processPendingBlocksPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot / 2)

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
	if _, ok := r.slotToPendingBlocks[block.Slot]; ok {
		return nil
	}

	// Handle block when the parent is unknown
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(block.ParentRoot)) {
		r.slotToPendingBlocksLock.Lock()
		r.seenPendingBlocksLock.Lock()
		defer r.slotToPendingBlocksLock.Unlock()
		defer r.seenPendingBlocksLock.Unlock()
		blockRoot, err := ssz.SigningRoot(block)
		if err != nil {
			log.Errorf("Could not sign root block: %v", err)
			return nil
		}
		r.slotToPendingBlocks[block.Slot] = block
		r.seenPendingBlocks[blockRoot] = true

		if !r.seenPendingBlocks[bytesutil.ToBytes32(block.ParentRoot)] {
			log.Warnf("Received a block from slot %d which we do not have the parent block in the database. "+
				"Requesting parent.", block.Slot)
		}

		return nil
	}

	go r.processPendingBlocksPoll(ctx)

	return r.chain.ReceiveBlockNoPubsub(ctx, block)
}

// processes pending blocks every processPendingBlocksPeriod
func (r *RegularSync) processPendingBlocksPoll(ctx context.Context) {
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

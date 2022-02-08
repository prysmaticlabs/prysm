package blockchain

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

// optimisticCandidateBlock returns true if this block can be optimistically synced.
//
// Spec pseudocode definition:
// def is_optimistic_candidate_block(opt_store: OptimisticStore, current_slot: Slot, block: BeaconBlock) -> bool:
//     justified_root = opt_store.block_states[opt_store.head_block_root].current_justified_checkpoint.root
//     justified_is_execution_block = is_execution_block(opt_store.blocks[justified_root])
//     block_is_deep = block.slot + SAFE_SLOTS_TO_IMPORT_OPTIMISTICALLY <= current_slot
//     return justified_is_execution_block or block_is_deep
func (s *Service) optimisticCandidateBlock(ctx context.Context, blk block.BeaconBlock) (bool, error) {
	if blk.Slot()+params.BeaconConfig().SafeSlotsToImportOptimistically <= s.CurrentSlot() {
		return true, nil
	}
	j := s.store.JustifiedCheckpt()
	if j == nil {
		return false, errNilJustifiedInStore
	}
	jBlock, err := s.cfg.BeaconDB.Block(ctx, bytesutil.ToBytes32(j.Root))
	if err != nil {
		return false, err
	}
	return blocks.ExecutionBlock(jBlock.Block().Body())
}

// removeInvalidChain removes all the ancestors of the given INVALID block
// from DB and ForkChoice until we reach a known VALID block.
// Warning: this method does not remove the given block itself which is
// assumed not to be stored in ForkChoice nor in the DB
func (s *Service) removeInvalidChain(ctx context.Context, b block.BeaconBlock) error {

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		parentRoot := bytesutil.ToBytes32(b.ParentRoot())
		parent, err := s.cfg.BeaconDB.Block(s.ctx, parentRoot)
		if err != nil {
			return err
		}
		payload, err := parent.Block().Body().ExecutionPayload()
		if err != nil {
			return err
		}
		_, err = s.cfg.ExecutionEngineCaller.ExecutePayload(ctx, executionPayloadToExecutableData(payload))
		if err != powchain.ErrInvalidPayload {
			return nil
		}
		err = s.cfg.ForkChoiceStore.UpdateSyncedTipsWithInvalidRoot(ctx, parentRoot)
		if err != nil {
			return err
		}
		err = s.cfg.BeaconDB.DeleteBlock(ctx, parentRoot)
		if err != nil {
			if err == kv.ErrDeleteFinalized {
				panic(err)
			}
			return err
		}
		log.WithField("BeaconBlockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(parentRoot[:]))).Info(
			"Removed block with invalid execution payload")
		b = parent.Block()
	}
}

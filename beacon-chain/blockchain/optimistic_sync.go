package blockchain

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
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

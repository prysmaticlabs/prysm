package helpers

import (
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BlockRootAtSlot returns the block root stored in the BeaconState for a recent slot.
// It returns an error if the requested block root is not within the slot range.
// Spec pseudocode definition:
// 	def get_block_root_at_slot(state: BeaconState,
//                           slot: Slot) -> Bytes32:
//    """
//    Return the block root at a recent ``slot``.
//    """
//    assert slot < state.slot <= slot + SLOTS_PER_HISTORICAL_ROOT
//    return state.latest_block_roots[slot % SLOTS_PER_HISTORICAL_ROOT]
func BlockRootAtSlot(state *pb.BeaconState, slot uint64) ([]byte, error) {
	earliestSlot := uint64(0)
	if state.Slot > params.BeaconConfig().SlotsPerHistoricalRoot {
		earliestSlot = state.Slot - params.BeaconConfig().SlotsPerHistoricalRoot
	}
	if slot < earliestSlot || slot >= state.Slot {
		return []byte{}, fmt.Errorf("slot %d is not within range %d to %d",
			slot,
			earliestSlot,
			state.Slot,
		)
	}
	return state.LatestBlockRoots[slot%params.BeaconConfig().SlotsPerHistoricalRoot], nil
}

// BlockRoot returns the block root stored in the BeaconState for epoch start slot.
// 	def get_block_root(state: BeaconState,
//                   epoch: Epoch) -> Bytes32:
//    """
//    Return the block root at a recent ``epoch``.
//    """
//    return get_block_root_at_slot(state, get_epoch_start_slot(epoch))
func BlockRoot(state *pb.BeaconState, epoch uint64) ([]byte, error) {
	return BlockRootAtSlot(state, StartSlot(epoch))
}

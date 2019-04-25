package helpers

import (
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BlockRoot returns the block root stored in the BeaconState for a recent slot.
// It returns an error if the requested block root is not within the slot range.
// Spec pseudocode definition:
// 	def get_block_root(state: BeaconState,
//                   slot: Slot) -> Bytes32:
//    """
//    Return the block root at a recent ``slot``.
//    """
//    assert slot < state.slot <= slot + SLOTS_PER_HISTORICAL_ROOT
//    return state.latest_block_roots[slot % SLOTS_PER_HISTORICAL_ROOT]
func BlockRoot(state *pb.BeaconState, slot uint64) ([]byte, error) {
	earliestSlot := state.Slot - params.BeaconConfig().SlotsPerHistoricalRoot

	if slot < earliestSlot || slot >= state.Slot {
		if earliestSlot < params.BeaconConfig().GenesisSlot {
			earliestSlot = params.BeaconConfig().GenesisSlot
		}
		return []byte{}, fmt.Errorf("slot %d is not within range %d to %d",
			slot-params.BeaconConfig().GenesisSlot,
			earliestSlot-params.BeaconConfig().GenesisSlot,
			state.Slot-params.BeaconConfig().GenesisSlot,
		)
	}
	return state.LatestBlockRoots[slot%params.BeaconConfig().SlotsPerHistoricalRoot], nil
}

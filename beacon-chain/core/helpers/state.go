package helpers

import (
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// StateRoot returns the state root stored in the BeaconState for a recent slot.
// It returns an error if the requested state root is not within the slot range.
// Spec pseudocode definition:
// 	def get_state_root(state: BeaconState,
//                   slot: Slot) -> Bytes32:
//    """
//    Return the state root at a recent ``slot``.
//    """
//    assert slot < state.slot <= slot + SLOTS_PER_HISTORICAL_ROOT
//    return state.latest_state_roots[slot % SLOTS_PER_HISTORICAL_ROOT]
func StateRoot(state *pb.BeaconState, slot uint64) ([]byte, error) {
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
	return state.LatestStateRoots[slot%params.BeaconConfig().SlotsPerHistoricalRoot], nil
}

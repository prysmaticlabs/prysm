package helpers

import (

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// RandaoMix returns the randao mix of a given epoch.
//
// Spec pseudocode definition:
//   def get_active_index_root(state: BeaconState,
//                          epoch: EpochNumber) -> Bytes32:
//    """
//    Return the index root at a recent ``epoch``.
//    """
//    assert get_current_epoch(state) - LATEST_INDEX_ROOTS_LENGTH + ENTRY_EXIT_DELAY < epoch <= get_current_epoch(state) + ENTRY_EXIT_DELAY
//    return state.latest_index_roots[epoch % LATEST_INDEX_ROOTS_LENGTH]
func RandaoMix(state *pb.BeaconState, epoch uint64) [32]byte{

}
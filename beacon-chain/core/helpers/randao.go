package helpers

import (
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// RandaoMix returns the randao mix of a given epoch.
//
// Spec pseudocode definition:
//   def generate_seed(state: BeaconState,
//                  epoch: EpochNumber) -> Bytes32:
//    """
//    Generate a seed for the given ``epoch``.
//    """
//    return hash(
//        get_randao_mix(state, epoch - SEED_LOOKAHEAD) +
//        get_active_index_root(state, epoch)
//    )
func GenerateSeed(state *pb.BeaconState, epoch uint64) [32]byte{

	hashutil.Hash()
}

// RandaoMix returns the randao mix (xor'ed seed)
// of a given slot. It is used to shuffle validators.
//
// Spec pseudocode definition:
//   def get_randao_mix(state: BeaconState,
//                   epoch: EpochNumber) -> Bytes32:
//    """
//    Return the randao mix at a recent ``epoch``.
//    """
//    assert get_current_epoch(state) - LATEST_RANDAO_MIXES_LENGTH < epoch <= get_current_epoch(state)
//    return state.latest_randao_mixes[epoch % LATEST_RANDAO_MIXES_LENGTH]
func RandaoMix(state *pb.BeaconState, epoch uint64) ([]byte, error) {
	var lowerBound uint64
	if state.Slot > config.LatestRandaoMixesLength {
		lowerBound = state.Slot - config.LatestRandaoMixesLength
	}
	upperBound := state.Slot
	if lowerBound > slot || slot >= upperBound {
		return nil, fmt.Errorf("input randaoMix slot %d out of bounds: %d <= slot < %d",
			slot, lowerBound, upperBound)
	}
	return state.LatestRandaoMixesHash32S[slot%config.LatestRandaoMixesLength], nil
}
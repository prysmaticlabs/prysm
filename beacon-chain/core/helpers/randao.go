package helpers

import (
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Seed returns the randao seed used for shuffling of a given epoch.
//
// Spec pseudocode definition:
//  def get_seed(state: BeaconState, epoch: Epoch, domain_type: DomainType) -> Hash:
//    """
//    Return the seed at ``epoch``.
//    """
//    mix = get_randao_mix(state, Epoch(epoch + EPOCHS_PER_HISTORICAL_VECTOR - MIN_SEED_LOOKAHEAD - 1))  # Avoid underflow
//    return hash(domain_type + int_to_bytes(epoch, length=8) + mix)
func Seed(state *stateTrie.BeaconState, epoch uint64, domain [bls.DomainByteLength]byte) ([32]byte, error) {
	// See https://github.com/ethereum/eth2.0-specs/pull/1296 for
	// rationale on why offset has to look down by 1.
	lookAheadEpoch := epoch + params.BeaconConfig().EpochsPerHistoricalVector -
		params.BeaconConfig().MinSeedLookahead - 1

	randaoMix, err := RandaoMix(state, lookAheadEpoch)
	if err != nil {
		return [32]byte{}, err
	}
	seed := append(domain[:], bytesutil.Bytes8(epoch)...)
	seed = append(seed, randaoMix...)

	seed32 := hashutil.Hash(seed)

	return seed32, nil
}

// RandaoMix returns the randao mix (xor'ed seed)
// of a given slot. It is used to shuffle validators.
//
// Spec pseudocode definition:
//   def get_randao_mix(state: BeaconState, epoch: Epoch) -> Hash:
//    """
//    Return the randao mix at a recent ``epoch``.
//    """
//    return state.randao_mixes[epoch % EPOCHS_PER_HISTORICAL_VECTOR]
func RandaoMix(state *stateTrie.BeaconState, epoch uint64) ([]byte, error) {
	return state.RandaoMixAtIndex(epoch % params.BeaconConfig().EpochsPerHistoricalVector)
}

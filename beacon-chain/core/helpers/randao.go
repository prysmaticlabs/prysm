package helpers

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var currentEpochSeed = cache.NewSeedCache()

// ErrInvalidStateLatestActiveIndexRoots is returned when the state active
// index root count does not match the expected EpochsPerHistoricalVector.
var ErrInvalidStateLatestActiveIndexRoots = errors.New("state does not have correct number of latest active index roots")

// Seed returns the randao seed used for shuffling of a given epoch.
//
// Spec pseudocode definition:
//  def get_seed(state: BeaconState, epoch: Epoch) -> Hash:
//    """
//    Return the seed at ``epoch``.
//    """
//    mix = get_randao_mix(state, Epoch(epoch + EPOCHS_PER_HISTORICAL_VECTOR - MIN_SEED_LOOKAHEAD - 1)) #Avoid underflow
//    active_index_root = state.active_index_roots[epoch % EPOCHS_PER_HISTORICAL_VECTOR]
//    return hash(mix + active_index_root + int_to_bytes(epoch, length=32))
func Seed(state *pb.BeaconState, epoch uint64) ([32]byte, error) {
	seed, err := currentEpochSeed.SeedInEpoch(epoch)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not retrieve total balance from cache")
	}
	if seed != nil {
		return bytesutil.ToBytes32(seed), nil
	}

	// See https://github.com/ethereum/eth2.0-specs/pull/1296 for
	// rationale on why offset has to look down by 1.
	lookAheadEpoch := epoch + params.BeaconConfig().EpochsPerHistoricalVector -
		params.BeaconConfig().MinSeedLookahead - 1

	// Check that the state has the correct latest active index roots or
	// randao mix may panic for index out of bounds.
	if uint64(len(state.ActiveIndexRoots)) != params.BeaconConfig().EpochsPerHistoricalVector {
		return [32]byte{}, ErrInvalidStateLatestActiveIndexRoots
	}
	randaoMix := RandaoMix(state, lookAheadEpoch)

	indexRoot := ActiveIndexRoot(state, epoch)

	th := append(randaoMix, indexRoot...)
	th = append(th, bytesutil.Bytes32(epoch)...)

	seed32 := hashutil.Hash(th)

	if err := currentEpochSeed.AddSeed(&cache.SeedByEpoch{
		Epoch: epoch,
		Seed:  seed32[:],
	}); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not save active balance for cache")
	}

	return seed32, nil
}

// ActiveIndexRoot returns the index root of a given epoch.
//
// Spec pseudocode definition:
//   def get_active_index_root(state: BeaconState,
//                          epoch: Epoch) -> Bytes32:
//    """
//    Return the index root at a recent ``epoch``.
//    ``epoch`` expected to be between
//    (current_epoch - LATEST_ACTIVE_INDEX_ROOTS_LENGTH + ACTIVATION_EXIT_DELAY, current_epoch + ACTIVATION_EXIT_DELAY].
//    """
//    return state.latest_active_index_roots[epoch % LATEST_ACTIVE_INDEX_ROOTS_LENGTH]
func ActiveIndexRoot(state *pb.BeaconState, epoch uint64) []byte {
	newRootLength := len(state.ActiveIndexRoots[epoch%params.BeaconConfig().EpochsPerHistoricalVector])
	newRoot := make([]byte, newRootLength)
	copy(newRoot, state.ActiveIndexRoots[epoch%params.BeaconConfig().EpochsPerHistoricalVector])
	return newRoot
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
func RandaoMix(state *pb.BeaconState, epoch uint64) []byte {
	newMixLength := len(state.RandaoMixes[epoch%params.BeaconConfig().EpochsPerHistoricalVector])
	newMix := make([]byte, newMixLength)
	copy(newMix, state.RandaoMixes[epoch%params.BeaconConfig().EpochsPerHistoricalVector])
	return newMix
}

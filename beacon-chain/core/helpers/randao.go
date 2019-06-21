package helpers

import (
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var currentEpochSeed = cache.NewSeedCache()

// GenerateSeed generates the randao seed of a given epoch.
//
// Spec pseudocode definition:
//  def generate_seed(state: BeaconState,
//     epoch: Epoch) -> Bytes32:
//     """
//     Generate a seed for the given ``epoch``.
//     """
//     return hash(
//     get_randao_mix(state, epoch + LATEST_RANDAO_MIXES_LENGTH - MIN_SEED_LOOKAHEAD) +
//     get_active_index_root(state, epoch) +
//     int_to_bytes32(epoch)
// )
func GenerateSeed(state *pb.BeaconState, epoch uint64) ([32]byte, error) {
	seed, err := currentEpochSeed.SeedInEpoch(epoch)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not retrieve total balance from cache: %v", err)
	}
	if seed != nil {
		return bytesutil.ToBytes32(seed), nil
	}

	lookAheadEpoch := epoch + params.BeaconConfig().LatestRandaoMixesLength -
		params.BeaconConfig().MinSeedLookahead

	randaoMix := RandaoMix(state, lookAheadEpoch)

	indexRoot := ActiveIndexRoot(state, epoch)

	th := append(randaoMix, indexRoot...)
	th = append(th, bytesutil.Bytes32(epoch)...)

	seed32 := hashutil.Hash(th)

	if err := currentEpochSeed.AddSeed(&cache.SeedByEpoch{
		Epoch: epoch,
		Seed:  seed32[:],
	}); err != nil {
		return [32]byte{}, fmt.Errorf("could not save active balance for cache: %v", err)
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
	return state.LatestActiveIndexRoots[epoch%params.BeaconConfig().LatestActiveIndexRootsLength]
}

// RandaoMix returns the randao mix (xor'ed seed)
// of a given slot. It is used to shuffle validators.
//
// Spec pseudocode definition:
//   def get_randao_mix(state: BeaconState,
//                   epoch: Epoch) -> Bytes32:
//    """
//    Return the randao mix at a recent ``epoch``.
//    ``epoch`` expected to be between (current_epoch - LATEST_RANDAO_MIXES_LENGTH, current_epoch].
//    """
//    return state.latest_randao_mixes[epoch % LATEST_RANDAO_MIXES_LENGTH]
func RandaoMix(state *pb.BeaconState, epoch uint64) []byte {
	return state.LatestRandaoMixes[epoch%params.BeaconConfig().LatestRandaoMixesLength]
}

// CreateRandaoReveal generates a epoch signature using the beacon proposer priv key.
func CreateRandaoReveal(beaconState *pb.BeaconState, epoch uint64, privKeys []*bls.SecretKey) ([]byte, error) {
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := BeaconProposerIndex(beaconState)
	if err != nil {
		return []byte{}, fmt.Errorf("could not get beacon proposer index: %v", err)
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := Domain(beaconState, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	return epochSignature.Marshal(), nil
}

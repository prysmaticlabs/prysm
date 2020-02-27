package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const bytesPerChunk = 32
const cacheSize = 100000

var nocachedHasher *stateRootHasher
var cachedHasher *stateRootHasher

func init() {
	rootsCache, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: cacheSize, // number of keys to track frequency of (1M).
		MaxCost:     1 << 22,   // maximum cost of cache (3MB).
		// 100,000 roots will take up approximately 3 MB in memory.
		BufferItems: 64, // number of keys per Get buffer.
	})
	// Temporarily disable roots cache until cache issues can be resolved.
	cachedHasher = &stateRootHasher{rootsCache: rootsCache}
	nocachedHasher = &stateRootHasher{}
}

type stateRootHasher struct {
	rootsCache *ristretto.Cache
}

// HashTreeRootState provides a fully-customized version of ssz.HashTreeRoot
// for the BeaconState type of the official Ethereum Serenity specification.
// The reason for this particular function is to optimize for speed and memory allocation
// at the expense of complete specificity (that is, this function can only be used
// on the Prysm BeaconState data structure).
func HashTreeRootState(state *pb.BeaconState) ([32]byte, error) {
	if featureconfig.Get().EnableSSZCache {
		return cachedHasher.hashTreeRootState(state)
	}
	return nocachedHasher.hashTreeRootState(state)
}

// ComputeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
func ComputeFieldRoots(state *pb.BeaconState) ([][]byte, error) {
	if featureconfig.Get().EnableSSZCache {
		return cachedHasher.computeFieldRoots(state)
	}
	return nocachedHasher.computeFieldRoots(state)
}

func (h *stateRootHasher) hashTreeRootState(state *pb.BeaconState) ([32]byte, error) {
	var fieldRoots [][]byte
	var err error
	if featureconfig.Get().EnableSSZCache {
		fieldRoots, err = cachedHasher.computeFieldRoots(state)
		if err != nil {
			return [32]byte{}, err
		}
	} else {
		fieldRoots, err = nocachedHasher.computeFieldRoots(state)
		if err != nil {
			return [32]byte{}, err
		}
	}
	return bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func (h *stateRootHasher) computeFieldRoots(state *pb.BeaconState) ([][]byte, error) {
	if state == nil {
		return nil, errors.New("nil state")
	}
	// There are 20 fields in the beacon state.
	fieldRoots := make([][]byte, 20)

	// Genesis time root.
	genesisRoot := Uint64Root(state.GenesisTime)
	fieldRoots[0] = genesisRoot[:]

	// Slot root.
	slotRoot := Uint64Root(state.Slot)
	fieldRoots[1] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := ForkRoot(state.Fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[2] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := BlockHeaderRoot(state.LatestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[3] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRootsRoot, err := h.arraysRoot(state.BlockRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "BlockRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[4] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := h.arraysRoot(state.StateRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "StateRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[5] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	historicalRootsRt, err := HistoricalRootsRoot(state.HistoricalRoots)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[6] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := Eth1Root(state.Eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[7] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := Eth1DataVotesRoot(state.Eth1DataVotes)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[8] = eth1VotesRoot[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.Eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[9] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoot, err := h.validatorRegistryRoot(state.Validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[10] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := ValidatorBalancesRoot(state.Balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[11] = balancesRoot[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := h.arraysRoot(state.RandaoMixes, params.BeaconConfig().EpochsPerHistoricalVector, "RandaoMixes")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[12] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := SlashingsRoot(state.Slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[13] = slashingsRootsRoot[:]

	// PreviousEpochAttestations slice root.
	prevAttsRoot, err := h.epochAttestationsRoot(state.PreviousEpochAttestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	fieldRoots[14] = prevAttsRoot[:]

	// CurrentEpochAttestations slice root.
	currAttsRoot, err := h.epochAttestationsRoot(state.CurrentEpochAttestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	fieldRoots[15] = currAttsRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits)
	fieldRoots[16] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := CheckpointRoot(state.PreviousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[17] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := CheckpointRoot(state.CurrentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[18] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := CheckpointRoot(state.FinalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[19] = finalRoot[:]
	return fieldRoots, nil
}

// Uint64Root computes the HashTreeRoot Merkleization of
// a simple uint64 value according to the eth2
// Simple Serialize specification.
func Uint64Root(val uint64) [32]byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, val)
	root := bytesutil.ToBytes32(buf)
	return root
}

// ForkRoot computes the HashTreeRoot Merkleization of
// a Fork struct value according to the eth2
// Simple Serialize specification.
func ForkRoot(fork *pb.Fork) ([32]byte, error) {
	fieldRoots := make([][]byte, 3)
	if fork != nil {
		prevRoot := bytesutil.ToBytes32(fork.PreviousVersion)
		fieldRoots[0] = prevRoot[:]
		currRoot := bytesutil.ToBytes32(fork.CurrentVersion)
		fieldRoots[1] = currRoot[:]
		forkEpochBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(forkEpochBuf, fork.Epoch)
		epochRoot := bytesutil.ToBytes32(forkEpochBuf)
		fieldRoots[2] = epochRoot[:]
	}
	return bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// CheckpointRoot computes the HashTreeRoot Merkleization of
// a Checkpoint struct value according to the eth2
// Simple Serialize specification.
func CheckpointRoot(checkpoint *ethpb.Checkpoint) ([32]byte, error) {
	fieldRoots := make([][]byte, 2)
	if checkpoint != nil {
		epochBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(epochBuf, checkpoint.Epoch)
		epochRoot := bytesutil.ToBytes32(epochBuf)
		fieldRoots[0] = epochRoot[:]
		ckpRoot := bytesutil.ToBytes32(checkpoint.Root)
		fieldRoots[1] = ckpRoot[:]
	}
	return bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// HistoricalRootsRoot computes the HashTreeRoot Merkleization of
// a list of [32]byte historical block roots according to the eth2
// Simple Serialize specification.
func HistoricalRootsRoot(historicalRoots [][]byte) ([32]byte, error) {
	result, err := bitwiseMerkleize(historicalRoots, uint64(len(historicalRoots)), params.BeaconConfig().HistoricalRootsLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	historicalRootsBuf := new(bytes.Buffer)
	if err := binary.Write(historicalRootsBuf, binary.LittleEndian, uint64(len(historicalRoots))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal historical roots length")
	}
	// We need to mix in the length of the slice.
	historicalRootsOutput := make([]byte, 32)
	copy(historicalRootsOutput, historicalRootsBuf.Bytes())
	mixedLen := mixInLength(result, historicalRootsOutput)
	return mixedLen, nil
}

// SlashingsRoot computes the HashTreeRoot Merkleization of
// a list of uint64 slashing values according to the eth2
// Simple Serialize specification.
func SlashingsRoot(slashings []uint64) ([32]byte, error) {
	slashingMarshaling := make([][]byte, params.BeaconConfig().EpochsPerSlashingsVector)
	for i := 0; i < len(slashings) && i < len(slashingMarshaling); i++ {
		slashBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slashBuf, slashings[i])
		slashingMarshaling[i] = slashBuf
	}
	slashingChunks, err := pack(slashingMarshaling)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not pack slashings into chunks")
	}
	return bitwiseMerkleize(slashingChunks, uint64(len(slashingChunks)), uint64(len(slashingChunks)))
}

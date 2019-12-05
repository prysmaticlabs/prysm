package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const bytesPerChunk = 32
const cacheSize = 100000

var (
	rootsCache  *ristretto.Cache
	leavesCache = make(map[string][][]byte)
	layersCache = make(map[string][][][]byte)
)

func init() {
	rootsCache, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: cacheSize, // number of keys to track frequency of (1M).
		MaxCost:     1 << 22,   // maximum cost of cache (3MB).
		// 100,000 roots will take up approximately 3 MB in memory.
		BufferItems: 64, // number of keys per Get buffer.
	})
}

// HashTreeRootState provides a fully-customized version of ssz.HashTreeRoot
// for the BeaconState type of the official Ethereum Serenity specification.
// The reason for this particular function is to optimize for speed and memory allocation
// at the expense of complete specificity (that is, this function can only be used
// on the Prysm BeaconState data structure).
func HashTreeRootState(state *pb.BeaconState) ([32]byte, error) {
	// There are 20 fields in the beacon state.
	fieldRoots := make([][]byte, 20)

	// Genesis time root.
	genesisBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(genesisBuf, state.GenesisTime)
	genesisBufRoot := bytesutil.ToBytes32(genesisBuf)
	fieldRoots[0] = genesisBufRoot[:]

	// Slot root.
	slotBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBuf, state.Slot)
	slotBufRoot := bytesutil.ToBytes32(slotBuf)
	fieldRoots[1] = slotBufRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := forkRoot(state.Fork)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[2] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := blockHeaderRoot(state.LatestBlockHeader)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[3] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRootsRoot, err := ArraysRoot(state.BlockRoots, "BlockRoots")
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[4] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := ArraysRoot(state.StateRoots, "StateRoots")
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[5] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	historicalRootsRt, err := historicalRootsRoot(state.HistoricalRoots)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[6] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := eth1Root(state.Eth1Data)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[7] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := eth1DataVotesRoot(state.Eth1DataVotes)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[8] = eth1VotesRoot[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.Eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[9] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoot, err := validatorRegistryRoot(state.Validators)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[10] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := validatorBalancesRoot(state.Balances)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[11] = balancesRoot[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := ArraysRoot(state.RandaoMixes, "RandaoMixes")
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[12] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := slashingsRoot(state.Slashings)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[13] = slashingsRootsRoot[:]

	// PreviousEpochAttestations slice root.
	prevAttsRoot, err := epochAttestationsRoot(state.PreviousEpochAttestations)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	fieldRoots[14] = prevAttsRoot[:]

	// CurrentEpochAttestations slice root.
	currAttsRoot, err := epochAttestationsRoot(state.CurrentEpochAttestations)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	fieldRoots[15] = currAttsRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits)
	fieldRoots[16] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := checkpointRoot(state.PreviousJustifiedCheckpoint)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[17] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := checkpointRoot(state.CurrentJustifiedCheckpoint)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[18] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := checkpointRoot(state.FinalizedCheckpoint)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[19] = finalRoot[:]
	//
	//for i := 0; i < len(fieldRoots); i++ {
	//	fmt.Printf("%#x and index %d\n", fieldRoots[i], i)
	//}

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute full beacon state merkleization")
	}
	return root, nil
}

func forkRoot(fork *pb.Fork) ([32]byte, error) {
	fieldRoots := make([][]byte, 3)
	prevRoot := bytesutil.ToBytes32(fork.PreviousVersion)
	fieldRoots[0] = prevRoot[:]
	currRoot := bytesutil.ToBytes32(fork.CurrentVersion)
	fieldRoots[1] = currRoot[:]
	forkEpochBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(forkEpochBuf, fork.Epoch)
	epochRoot := bytesutil.ToBytes32(forkEpochBuf)
	fieldRoots[2] = epochRoot[:]
	return bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func checkpointRoot(checkpoint *ethpb.Checkpoint) ([32]byte, error) {
	fieldRoots := make([][]byte, 2)
	if checkpoint != nil {
		epochBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(epochBuf, checkpoint.Epoch)
		epochRoot := bytesutil.ToBytes32(epochBuf)
		fieldRoots[0] = epochRoot[:]
		fieldRoots[1] = checkpoint.Root
	}
	return bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func historicalRootsRoot(historicalRoots [][]byte) ([32]byte, error) {
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

func slashingsRoot(slashings []uint64) ([32]byte, error) {
	slashingMarshaling := make([][]byte, params.BeaconConfig().EpochsPerSlashingsVector)
	for i := 0; i < len(slashingMarshaling); i++ {
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

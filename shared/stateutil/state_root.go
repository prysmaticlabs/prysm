package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const bytesPerChunk = 32

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
	blockRootsRoot, err := bitwiseMerkleize(state.BlockRoots, uint64(len(state.BlockRoots)), uint64(len(state.BlockRoots)))
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[4] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := bitwiseMerkleize(state.StateRoots, uint64(len(state.StateRoots)), uint64(len(state.StateRoots)))
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
	validatorsRoots := make([][]byte, 0)
	for i := 0; i < len(state.Validators); i++ {
		val, err := validatorRoot(state.Validators[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute validators merkleization")
		}
		validatorsRoots = append(validatorsRoots, val[:])
	}
	validatorsRootsRoot, err := bitwiseMerkleize(validatorsRoots, uint64(len(validatorsRoots)), params.BeaconConfig().ValidatorRegistryLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	validatorsRootsBuf := new(bytes.Buffer)
	if err := binary.Write(validatorsRootsBuf, binary.LittleEndian, uint64(len(state.Validators))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal validator registry length")
	}
	// We need to mix in the length of the slice.
	validatorsRootsBufRoot := make([]byte, 32)
	copy(validatorsRootsBufRoot, validatorsRootsBuf.Bytes())
	mixedValLen := mixInLength(validatorsRootsRoot, validatorsRootsBufRoot)
	fieldRoots[10] = mixedValLen[:]

	// Balances slice root.
	balancesMarshaling := make([][]byte, 0)
	for i := 0; i < len(state.Balances); i++ {
		balanceBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(balanceBuf, state.Balances[i])
		balancesMarshaling = append(balancesMarshaling, balanceBuf)
	}
	balancesChunks, err := pack(balancesMarshaling)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not pack balances into chunks")
	}
	maxBalCap := params.BeaconConfig().ValidatorRegistryLimit
	elemSize := uint64(8)
	balLimit := (maxBalCap*elemSize + 31) / 32
	if balLimit == 0 {
		if len(state.Balances) == 0 {
			balLimit = 1
		} else {
			balLimit = uint64(len(state.Balances))
		}
	}
	balancesRootsRoot, err := bitwiseMerkleize(balancesChunks, uint64(len(balancesChunks)), balLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute balances merkleization")
	}
	balancesRootsBuf := new(bytes.Buffer)
	if err := binary.Write(balancesRootsBuf, binary.LittleEndian, uint64(len(state.Balances))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal balances length")
	}
	balancesRootsBufRoot := make([]byte, 32)
	copy(balancesRootsBufRoot, balancesRootsBuf.Bytes())
	mixedBalLen := mixInLength(balancesRootsRoot, balancesRootsBufRoot)
	fieldRoots[11] = mixedBalLen[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := bitwiseMerkleize(state.RandaoMixes, uint64(len(state.RandaoMixes)), uint64(len(state.RandaoMixes)))
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[12] = randaoRootsRoot[:]

	// Slashings array root.
	slashingMarshaling := make([][]byte, params.BeaconConfig().EpochsPerSlashingsVector)
	for i := 0; i < len(slashingMarshaling); i++ {
		slashBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slashBuf, state.Slashings[i])
		slashingMarshaling[i] = slashBuf
	}
	slashingChunks, err := pack(slashingMarshaling)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not pack slashings into chunks")
	}
	slashingRootsRoot, err := bitwiseMerkleize(slashingChunks, uint64(len(slashingChunks)), uint64(len(slashingChunks)))
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[13] = slashingRootsRoot[:]

	// PreviousEpochAttestations slice root.
	prevAttsRoots := make([][]byte, 0)
	for i := 0; i < len(state.PreviousEpochAttestations); i++ {
		pendingPrevRoot, err := pendingAttestationRoot(state.PreviousEpochAttestations[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not attestation merkleization")
		}
		prevAttsRoots = append(prevAttsRoots, pendingPrevRoot[:])
	}
	prevAttsRootsRoot, err := bitwiseMerkleize(prevAttsRoots, uint64(len(prevAttsRoots)), params.BeaconConfig().MaxAttestations*32)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	prevAttsLenBuf := new(bytes.Buffer)
	if err := binary.Write(prevAttsLenBuf, binary.LittleEndian, uint64(len(state.PreviousEpochAttestations))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal previous epoch attestations length")
	}
	// We need to mix in the length of the slice.
	prevAttsLenRoot := make([]byte, 32)
	copy(prevAttsLenRoot, prevAttsLenBuf.Bytes())
	prevRoot := mixInLength(prevAttsRootsRoot, prevAttsLenRoot)
	fieldRoots[14] = prevRoot[:]

	// CurrentEpochAttestations slice root.
	currAttsRoots := make([][]byte, 0)
	for i := 0; i < len(state.CurrentEpochAttestations); i++ {
		pendingRoot, err := pendingAttestationRoot(state.CurrentEpochAttestations[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not attestation merkleization")
		}
		currAttsRoots = append(currAttsRoots, pendingRoot[:])
	}
	currAttsRootsRoot, err := bitwiseMerkleize(currAttsRoots, uint64(len(currAttsRoots)), params.BeaconConfig().MaxAttestations*32)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute current epoch attestations merkleization")
	}
	// We need to mix in the length of the slice.
	currAttsLenBuf := new(bytes.Buffer)
	if err := binary.Write(currAttsLenBuf, binary.LittleEndian, uint64(len(state.CurrentEpochAttestations))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal current epoch attestations length")
	}
	currAttsLenRoot := make([]byte, 32)
	copy(currAttsLenRoot, currAttsLenBuf.Bytes())
	currRoot := mixInLength(currAttsRootsRoot, currAttsLenRoot)
	fieldRoots[15] = currRoot[:]

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
	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	return root, nil
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
	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, nil
	}
	return root, nil
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

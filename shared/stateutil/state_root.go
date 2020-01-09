package stateutil

import (
	"bytes"
	"encoding/binary"
	"encoding/json"

	"github.com/gogo/protobuf/proto"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const bytesPerChunk = 32
const cacheSize = 100000

var globalHasher *stateRootHasher

func init() {
	rootsCache, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: cacheSize, // number of keys to track frequency of (1M).
		MaxCost:     1 << 22,   // maximum cost of cache (3MB).
		// 100,000 roots will take up approximately 3 MB in memory.
		BufferItems: 64, // number of keys per Get buffer.
	})
	// Temporarily disable roots cache until cache issues can be resolved.
	//globalHasher = &stateRootHasher{rootsCache: rootsCache}
	_ = rootsCache
	globalHasher = &stateRootHasher{}
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
	return globalHasher.hashTreeRootState(state)
}

func CopyState2(state *pb.BeaconState) *pb.BeaconState {
	ot, _ := json.Marshal(state)
	st := &pb.BeaconState{}
	json.Unmarshal(ot, st)
	return st
}

func CopyState(state *pb.BeaconState) *pb.BeaconState {
	blockRoots := make([]bytesutil.Bytes32Array, len(state.BlockRoots))
	for i, r := range state.BlockRoots {
		blockRoots[i] = r
	}
	stateRoots := make([]bytesutil.Bytes32Array, len(state.StateRoots))
	for i, r := range state.StateRoots {
		stateRoots[i] = r
	}
	historicalRoots := make([]bytesutil.Bytes32Array, len(state.HistoricalRoots))
	for i, r := range state.HistoricalRoots {
		historicalRoots[i] = r
	}
	randaoMixes := make([]bytesutil.Bytes32Array, len(state.RandaoMixes))
	for i, r := range state.RandaoMixes {
		randaoMixes[i] = r
	}
	eth1DataVotes := make([]*ethpb.Eth1Data, len(state.Eth1DataVotes))
	for i, v := range state.Eth1DataVotes {
		eth1DataVotes[i] = proto.Clone(v).(*ethpb.Eth1Data)
	}
	validators := make([]*ethpb.Validator, len(state.Validators))
	for i, v := range state.Validators {
		validators[i] = proto.Clone(v).(*ethpb.Validator)
	}
	balances := make([]uint64, len(state.Balances))
	copy(balances, state.Balances)

	slashings := make([]uint64, len(state.Slashings))
	copy(slashings, state.Slashings)

	prevAtt := make([]*pb.PendingAttestation, len(state.PreviousEpochAttestations))
	for i, p := range state.PreviousEpochAttestations {
		prevAtt[i] = proto.Clone(p).(*pb.PendingAttestation)
	}
	currAtt := make([]*pb.PendingAttestation, len(state.CurrentEpochAttestations))
	for i, p := range state.CurrentEpochAttestations {
		prevAtt[i] = proto.Clone(p).(*pb.PendingAttestation)
	}
	justBits := make([]byte, state.JustificationBits.Len())
	copy(justBits, state.JustificationBits)

	return &pb.BeaconState{
		GenesisTime:                 state.GenesisTime,
		Slot:                        state.Slot,
		Fork:                        proto.Clone(state.Fork).(*pb.Fork),
		LatestBlockHeader:           proto.Clone(state.LatestBlockHeader).(*ethpb.BeaconBlockHeader),
		BlockRoots:                  blockRoots,
		StateRoots:                  stateRoots,
		HistoricalRoots:             historicalRoots,
		Eth1Data:                    proto.Clone(state.Eth1Data).(*ethpb.Eth1Data),
		Eth1DataVotes:               eth1DataVotes,
		Eth1DepositIndex:            state.Eth1DepositIndex,
		Validators:                  validators,
		Balances:                    balances,
		RandaoMixes:                 randaoMixes,
		Slashings:                   slashings,
		PreviousEpochAttestations:   prevAtt,
		CurrentEpochAttestations:    currAtt,
		JustificationBits:           justBits,
		PreviousJustifiedCheckpoint: proto.Clone(state.PreviousJustifiedCheckpoint).(*ethpb.Checkpoint),
		CurrentJustifiedCheckpoint:  proto.Clone(state.CurrentJustifiedCheckpoint).(*ethpb.Checkpoint),
		FinalizedCheckpoint:         proto.Clone(state.FinalizedCheckpoint).(*ethpb.Checkpoint),
	}
}

func (h *stateRootHasher) hashTreeRootState(state *pb.BeaconState) ([32]byte, error) {
	if state == nil {
		return [32]byte{}, errors.New("nil state")
	}
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
	blockRootsRoot, err := h.arraysRoot(state.BlockRoots, "BlockRoots")
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[4] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := h.arraysRoot(state.StateRoots, "StateRoots")
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[5] = stateRootsRoot[:]

	histRoots := make([][]byte, len(state.HistoricalRoots))
	for i, r := range state.HistoricalRoots {
		histRoots[i] = r[:]
	}
	// HistoricalRoots slice root.
	historicalRootsRt, err := historicalRootsRoot(histRoots)
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
	validatorsRoot, err := h.validatorRegistryRoot(state.Validators)
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
	randaoRootsRoot, err := h.arraysRoot(state.RandaoMixes, "RandaoMixes")
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
	prevAttsRoot, err := h.epochAttestationsRoot(state.PreviousEpochAttestations)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	fieldRoots[14] = prevAttsRoot[:]

	// CurrentEpochAttestations slice root.
	currAttsRoot, err := h.epochAttestationsRoot(state.CurrentEpochAttestations)
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

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute full beacon state merkleization")
	}
	return root, nil
}

func forkRoot(fork *pb.Fork) ([32]byte, error) {
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

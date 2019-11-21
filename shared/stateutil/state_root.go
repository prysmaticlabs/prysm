package stateutil

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/minio/sha256-simd"
	"github.com/protolambda/zssz/htr"
	"github.com/protolambda/zssz/merkle"
	"github.com/prysmaticlabs/go-bitfield"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const bytesPerChunk = 32

// HashTreeRootState provides a fully-customized version of ssz.HashTreeRoot
// for the BeaconState type of the official Ethereum Serenity specification.
// The reason for this particular function is to optimize for speed and memory allocation
// at the expense of complete specificity (that is, this function can only be used
// on the Prysm BeaconState data structure).
func HashTreeRootState(state *pb.BeaconState) [32]byte {
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
	forkHashTreeRoot := forkRoot(state.Fork)
	fieldRoots[2] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot := blockHeaderRoot(state.LatestBlockHeader)
	fieldRoots[3] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRoots := merkleize(state.BlockRoots)
	fieldRoots[4] = blockRoots[:]

	// StateRoots array root.
	stateRoots := merkleize(state.StateRoots)
	fieldRoots[5] = stateRoots[:]

	// HistoricalRoots slice root.
	historicalRootsRoot, err := bitwiseMerkleize(state.HistoricalRoots, uint64(len(state.HistoricalRoots)), params.BeaconConfig().HistoricalRootsLimit)
	if err != nil {
		panic(err)
	}
	historicalRootsBuf := new(bytes.Buffer)
	if err := binary.Write(historicalRootsBuf, binary.LittleEndian, uint64(len(state.HistoricalRoots))); err != nil {
		panic(err)
	}
	// We need to mix in the length of the slice.
	historicalRootsOutput := make([]byte, 32)
	copy(historicalRootsOutput, historicalRootsBuf.Bytes())
	mixedLen := mixInLength(historicalRootsRoot, historicalRootsOutput)
	fieldRoots[6] = mixedLen[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot := eth1Root(state.Eth1Data)
	fieldRoots[7] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoots := make([][]byte, 0)
	for i := 0; i < len(state.Eth1DataVotes); i++ {
		eth1 := eth1Root(state.Eth1DataVotes[i])
		eth1VotesRoots = append(eth1VotesRoots, eth1[:])
	}
	eth1VotesRootsRoot, err := bitwiseMerkleize(eth1VotesRoots, uint64(len(eth1VotesRoots)), params.BeaconConfig().SlotsPerEth1VotingPeriod)
	if err != nil {
		panic(err)
	}
	eth1VotesRootBuf := new(bytes.Buffer)
	if err := binary.Write(eth1VotesRootBuf, binary.LittleEndian, uint64(len(state.Eth1DataVotes))); err != nil {
		panic(err)
	}
	// We need to mix in the length of the slice.
	eth1VotesRootBufRoot := make([]byte, 32)
	copy(eth1VotesRootBufRoot, eth1VotesRootBuf.Bytes())
	mixedEth1Root := mixInLength(eth1VotesRootsRoot, eth1VotesRootBufRoot)
	fieldRoots[8] = mixedEth1Root[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.Eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[9] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoots := make([][]byte, 0)
	for i := 0; i < len(state.Validators); i++ {
		val := validatorRoot(state.Validators[i])
		validatorsRoots = append(validatorsRoots, val[:])
	}
	validatorsRootsRoot, err := bitwiseMerkleize(validatorsRoots, uint64(len(validatorsRoots)), params.BeaconConfig().ValidatorRegistryLimit)
	if err != nil {
		panic(err)
	}
	validatorsRootsBuf := new(bytes.Buffer)
	if err := binary.Write(validatorsRootsBuf, binary.LittleEndian, uint64(len(state.Validators))); err != nil {
		panic(err)
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
		panic(err)
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
		panic(err)
	}
	balancesRootsBuf := new(bytes.Buffer)
	if err := binary.Write(balancesRootsBuf, binary.LittleEndian, uint64(len(state.Balances))); err != nil {
		panic(err)
	}
	balancesRootsBufRoot := make([]byte, 32)
	copy(balancesRootsBufRoot, balancesRootsBuf.Bytes())
	mixedBalLen := mixInLength(balancesRootsRoot, balancesRootsBufRoot)
	fieldRoots[11] = mixedBalLen[:]

	// RandaoMixes array root.
	randaoRoots := merkleize(state.RandaoMixes)
	fieldRoots[12] = randaoRoots[:]

	// Slashings array root.
	slashingMarshaling := make([][]byte, params.BeaconConfig().EpochsPerSlashingsVector)
	for i := 0; i < len(slashingMarshaling); i++ {
		slashBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slashBuf, state.Slashings[i])
		slashingMarshaling[i] = slashBuf
	}
	slashingChunks, err := pack(slashingMarshaling)
	if err != nil {
		panic(err)
	}
	slashingRootsRoot, err := bitwiseMerkleize(slashingChunks, uint64(len(slashingChunks)), uint64(len(slashingChunks)))
	if err != nil {
		panic(err)
	}
	fieldRoots[13] = slashingRootsRoot[:]

	// PreviousEpochAttestations slice root.
	prevAttsRoots := make([][]byte, 0)
	for i := 0; i < len(state.PreviousEpochAttestations); i++ {
		pendingPrevRoot := pendingAttestationRoot(state.PreviousEpochAttestations[i])
		prevAttsRoots = append(prevAttsRoots, pendingPrevRoot[:])
	}
	prevAttsRootsRoot, err := bitwiseMerkleize(prevAttsRoots, uint64(len(prevAttsRoots)), params.BeaconConfig().MaxAttestations*params.BeaconConfig().SlotsPerEpoch)
	if err != nil {
		panic(err)
	}
	prevAttsLenBuf := new(bytes.Buffer)
	if err := binary.Write(prevAttsLenBuf, binary.LittleEndian, uint64(len(state.PreviousEpochAttestations))); err != nil {
		panic(err)
	}
	// We need to mix in the length of the slice.
	prevAttsLenRoot := make([]byte, 32)
	copy(prevAttsLenRoot, prevAttsLenBuf.Bytes())
	prevRoot := mixInLength(prevAttsRootsRoot, prevAttsLenRoot)
	fieldRoots[14] = prevRoot[:]

	// CurrentEpochAttestations slice root.
	currAttsRoots := make([][]byte, 0)
	for i := 0; i < len(state.CurrentEpochAttestations); i++ {
		pendingRoot := pendingAttestationRoot(state.CurrentEpochAttestations[i])
		currAttsRoots = append(currAttsRoots, pendingRoot[:])
	}
	currAttsRootsRoot, err := bitwiseMerkleize(currAttsRoots, uint64(len(currAttsRoots)), params.BeaconConfig().MaxAttestations*params.BeaconConfig().SlotsPerEpoch)
	if err != nil {
		panic(err)
	}
	// We need to mix in the length of the slice.
	currAttsLenBuf := new(bytes.Buffer)
	if err := binary.Write(currAttsLenBuf, binary.LittleEndian, uint64(len(state.CurrentEpochAttestations))); err != nil {
		panic(err)
	}
	currAttsLenRoot := make([]byte, 32)
	copy(currAttsLenRoot, currAttsLenBuf.Bytes())
	currRoot := mixInLength(currAttsRootsRoot, currAttsLenRoot)
	fieldRoots[15] = currRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits)
	fieldRoots[16] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot := checkpointRoot(state.PreviousJustifiedCheckpoint)
	fieldRoots[17] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot := checkpointRoot(state.CurrentJustifiedCheckpoint)
	fieldRoots[18] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot := checkpointRoot(state.FinalizedCheckpoint)
	fieldRoots[19] = finalRoot[:]

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		panic(err)
	}
	return root
}

func forkRoot(fork *pb.Fork) [32]byte {
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
		panic(err)
	}
	return root
}

func blockHeaderRoot(header *ethpb.BeaconBlockHeader) [32]byte {
	fieldRoots := make([][]byte, 5)
	headerSlotBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(headerSlotBuf, header.Slot)
	headerSlotRoot := bytesutil.ToBytes32(headerSlotBuf)
	fieldRoots[0] = headerSlotRoot[:]
	fieldRoots[1] = header.ParentRoot
	fieldRoots[2] = header.StateRoot
	fieldRoots[3] = header.BodyRoot
	signatureChunks, err := pack([][]byte{header.Signature})
	if err != nil {
		panic(err)
	}
	sigRoot, err := bitwiseMerkleize(signatureChunks, uint64(len(signatureChunks)), uint64(len(signatureChunks)))
	if err != nil {
		panic(err)
	}
	fieldRoots[4] = sigRoot[:]
	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		panic(err)
	}
	return root
}

func attestationDataRoot(data *ethpb.AttestationData) [32]byte {
	fieldRoots := make([][]byte, 5)

	// Slot.
	slotBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBuf, data.Slot)
	slotRoot := bytesutil.ToBytes32(slotBuf)
	fieldRoots[0] = slotRoot[:]

	// Index.
	indexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(indexBuf, data.Index)
	interRoot := bytesutil.ToBytes32(indexBuf)
	fieldRoots[1] = interRoot[:]

	// Beacon block root.
	fieldRoots[2] = data.BeaconBlockRoot

	// Source
	sourceRoot := checkpointRoot(data.Source)
	fieldRoots[3] = sourceRoot[:]

	// Target
	targetRoot := checkpointRoot(data.Target)
	fieldRoots[4] = targetRoot[:]

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		panic(err)
	}
	return root
}

func pendingAttestationRoot(att *pb.PendingAttestation) [32]byte {
	fieldRoots := make([][]byte, 4)

	// Bitfield.
	aggregationRoot, err := bitlistRoot(att.AggregationBits, 2048)
	if err != nil {
		panic(err)
	}
	fieldRoots[0] = aggregationRoot[:]

	// Attestation data.
	attDataRoot := attestationDataRoot(att.Data)
	fieldRoots[1] = attDataRoot[:]

	// Inclusion delay.
	inclusionBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(inclusionBuf, att.InclusionDelay)
	inclusionRoot := bytesutil.ToBytes32(inclusionBuf)
	fieldRoots[2] = inclusionRoot[:]

	// Proposer index.
	proposerBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(proposerBuf, att.ProposerIndex)
	proposerRoot := bytesutil.ToBytes32(proposerBuf)
	fieldRoots[3] = proposerRoot[:]

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		panic(err)
	}
	return root
}

func validatorRoot(validator *ethpb.Validator) [32]byte {
	fieldRoots := make([][]byte, 8)

	// Public key.
	pubKeyChunks, err := pack([][]byte{validator.PublicKey})
	if err != nil {
		panic(err)
	}
	pubKeyRoot, err := bitwiseMerkleize(pubKeyChunks, uint64(len(pubKeyChunks)), uint64(len(pubKeyChunks)))
	if err != nil {
		panic(err)
	}
	fieldRoots[0] = pubKeyRoot[:]

	// Withdrawal credentials.
	fieldRoots[1] = validator.WithdrawalCredentials

	// Effective balance.
	effectiveBalanceBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(effectiveBalanceBuf, validator.EffectiveBalance)
	effBalRoot := bytesutil.ToBytes32(effectiveBalanceBuf)
	fieldRoots[2] = effBalRoot[:]

	// Slashed.
	slashBuf := make([]byte, 1)
	if validator.Slashed {
		slashBuf[0] = uint8(1)
	} else {
		slashBuf[0] = uint8(0)
	}
	slashBufRoot := bytesutil.ToBytes32(slashBuf)
	fieldRoots[3] = slashBufRoot[:]

	// Activation eligibility epoch.
	activationEligibilityBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(activationEligibilityBuf, validator.ActivationEligibilityEpoch)
	activationEligibilityRoot := bytesutil.ToBytes32(activationEligibilityBuf)
	fieldRoots[4] = activationEligibilityRoot[:]

	// Activation epoch.
	activationBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(activationBuf, validator.ActivationEpoch)
	activationRoot := bytesutil.ToBytes32(activationBuf)
	fieldRoots[5] = activationRoot[:]

	// Exit epoch.
	exitBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(exitBuf, validator.ExitEpoch)
	exitBufRoot := bytesutil.ToBytes32(exitBuf)
	fieldRoots[6] = exitBufRoot[:]

	// Withdrawable epoch.
	withdrawalBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(withdrawalBuf, validator.WithdrawableEpoch)
	withdrawalBufRoot := bytesutil.ToBytes32(withdrawalBuf)
	fieldRoots[7] = withdrawalBufRoot[:]

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		panic(err)
	}
	return root
}

func eth1Root(eth1Data *ethpb.Eth1Data) [32]byte {
	fieldRoots := make([][]byte, 3)
	fieldRoots[0] = eth1Data.DepositRoot
	eth1DataCountBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DataCountBuf, eth1Data.DepositCount)
	eth1CountRoot := bytesutil.ToBytes32(eth1DataCountBuf)
	fieldRoots[1] = eth1CountRoot[:]
	fieldRoots[2] = eth1Data.BlockHash
	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		panic(err)
	}
	return root
}

func checkpointRoot(checkpoint *ethpb.Checkpoint) [32]byte {
	fieldRoots := make([][]byte, 2)
	epochBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBuf, checkpoint.Epoch)
	epochRoot := bytesutil.ToBytes32(epochBuf)
	fieldRoots[0] = epochRoot[:]
	fieldRoots[1] = checkpoint.Root
	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		panic(err)
	}
	return root
}

func bitlistRoot(bfield bitfield.Bitfield, maxCapacity uint64) ([32]byte, error) {
	limit := (maxCapacity + 255) / 256
	if bfield == nil || bfield.Len() == 0 {
		length := make([]byte, 32)
		root, err := bitwiseMerkleize([][]byte{}, 0, limit)
		if err != nil {
			return [32]byte{}, err
		}
		return mixInLength(root, length), nil
	}
	chunks, err := pack([][]byte{bfield.Bytes()})
	if err != nil {
		return [32]byte{}, err
	}
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, bfield.Len()); err != nil {
		return [32]byte{}, err
	}
	output := make([]byte, 32)
	copy(output, buf.Bytes())
	root, err := bitwiseMerkleize(chunks, uint64(len(chunks)), limit)
	if err != nil {
		return [32]byte{}, err
	}
	return mixInLength(root, output), nil
}

// Given ordered BYTES_PER_CHUNK-byte chunks, if necessary utilize zero chunks so that the
// number of chunks is a power of two, Merkleize the chunks, and return the root.
// Note that merkleize on a single chunk is simply that chunk, i.e. the identity
// when the number of chunks is one.
func bitwiseMerkleize(chunks [][]byte, count uint64, limit uint64) ([32]byte, error) {
	if count > limit {
		return [32]byte{}, errors.New("merkleizing list that is too large, over limit")
	}
	hasher := htr.HashFn(hashutil.Hash)
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	return merkle.Merkleize(hasher, count, limit, leafIndexer), nil
}

func pack(serializedItems [][]byte) ([][]byte, error) {
	areAllEmpty := true
	for _, item := range serializedItems {
		if !bytes.Equal(item, []byte{}) {
			areAllEmpty = false
			break
		}
	}
	// If there are no items, we return an empty chunk.
	if len(serializedItems) == 0 || areAllEmpty {
		emptyChunk := make([]byte, bytesPerChunk)
		return [][]byte{emptyChunk}, nil
	} else if len(serializedItems[0]) == bytesPerChunk {
		// If each item has exactly BYTES_PER_CHUNK length, we return the list of serialized items.
		return serializedItems, nil
	}
	// We flatten the list in order to pack its items into byte chunks correctly.
	orderedItems := []byte{}
	for _, item := range serializedItems {
		orderedItems = append(orderedItems, item...)
	}
	numItems := len(orderedItems)
	chunks := [][]byte{}
	for i := 0; i < numItems; i += bytesPerChunk {
		j := i + bytesPerChunk
		// We create our upper bound index of the chunk, if it is greater than numItems,
		// we set it as numItems itself.
		if j > numItems {
			j = numItems
		}
		// We create chunks from the list of items based on the
		// indices determined above.
		chunks = append(chunks, orderedItems[i:j])
	}
	// Right-pad the last chunk with zero bytes if it does not
	// have length bytesPerChunk.
	lastChunk := chunks[len(chunks)-1]
	for len(lastChunk) < bytesPerChunk {
		lastChunk = append(lastChunk, 0)
	}
	chunks[len(chunks)-1] = lastChunk
	return chunks, nil
}

func merkleize(chunks [][]byte) [32]byte {
	if len(chunks) == 1 {
		var root [32]byte
		copy(root[:], chunks[0])
		return root
	}
	for !isPowerOf2(len(chunks)) {
		chunks = append(chunks, make([]byte, bytesPerChunk))
	}
	hashLayer := chunks
	// We keep track of the hash layers of a Merkle trie until we reach
	// the top layer of length 1, which contains the single root element.
	//        [Root]      -> Top layer has length 1.
	//    [E]       [F]   -> This layer has length 2.
	// [A]  [B]  [C]  [D] -> The bottom layer has length 4 (needs to be a power of two).
	i := 1
	for len(hashLayer) > 1 {
		layer := [][]byte{}
		for i := 0; i < len(hashLayer); i += 2 {
			hashedChunk := hashutil.Hash(append(hashLayer[i], hashLayer[i+1]...))
			layer = append(layer, hashedChunk[:])
		}
		hashLayer = layer
		i++
	}
	var root [32]byte
	copy(root[:], hashLayer[0])
	return root
}

func isPowerOf2(n int) bool {
	return n != 0 && (n&(n-1)) == 0
}

func mixInLength(root [32]byte, length []byte) [32]byte {
	var hash [32]byte
	h := sha256.New()
	h.Write(root[:])
	h.Write(length)
	// The hash interface never returns an error, for that reason
	// we are not handling the error below. For reference, it is
	// stated here https://golang.org/pkg/hash/#Hash
	// #nosec G104
	h.Sum(hash[:0])
	return hash
}

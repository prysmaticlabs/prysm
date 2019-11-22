package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/minio/sha256-simd"
	"github.com/pkg/errors"
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
	historicalRootsRoot, err := bitwiseMerkleize(state.HistoricalRoots, uint64(len(state.HistoricalRoots)), params.BeaconConfig().HistoricalRootsLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	historicalRootsBuf := new(bytes.Buffer)
	if err := binary.Write(historicalRootsBuf, binary.LittleEndian, uint64(len(state.HistoricalRoots))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal historical roots length")
	}
	// We need to mix in the length of the slice.
	historicalRootsOutput := make([]byte, 32)
	copy(historicalRootsOutput, historicalRootsBuf.Bytes())
	mixedLen := mixInLength(historicalRootsRoot, historicalRootsOutput)
	fieldRoots[6] = mixedLen[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := eth1Root(state.Eth1Data)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[7] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoots := make([][]byte, 0)
	for i := 0; i < len(state.Eth1DataVotes); i++ {
		eth1, err := eth1Root(state.Eth1DataVotes[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
		}
		eth1VotesRoots = append(eth1VotesRoots, eth1[:])
	}
	eth1Chunks, err := pack(eth1VotesRoots)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not chunk eth1 votes roots")
	}
	eth1VotesRootsRoot, err := bitwiseMerkleize(eth1Chunks, uint64(len(eth1Chunks)), uint64(1024))
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	eth1VotesRootBuf := new(bytes.Buffer)
	if err := binary.Write(eth1VotesRootBuf, binary.LittleEndian, uint64(len(state.Eth1DataVotes))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal eth1data votes length")
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

func blockHeaderRoot(header *ethpb.BeaconBlockHeader) ([32]byte, error) {
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
		return [32]byte{}, nil
	}
	sigRoot, err := bitwiseMerkleize(signatureChunks, uint64(len(signatureChunks)), uint64(len(signatureChunks)))
	if err != nil {
		return [32]byte{}, nil
	}
	fieldRoots[4] = sigRoot[:]
	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, nil
	}
	return root, nil
}

func attestationDataRoot(data *ethpb.AttestationData) ([32]byte, error) {
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
	sourceRoot, err := checkpointRoot(data.Source)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute source checkpoint merkleization")
	}
	fieldRoots[3] = sourceRoot[:]

	// Target
	targetRoot, err := checkpointRoot(data.Target)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute target checkpoint merkleization")
	}
	fieldRoots[4] = targetRoot[:]

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, nil
	}
	return root, nil
}

func pendingAttestationRoot(att *pb.PendingAttestation) ([32]byte, error) {
	fieldRoots := make([][]byte, 4)

	// Bitfield.
	aggregationRoot, err := bitlistRoot(att.AggregationBits, 2048)
	if err != nil {
		panic(err)
	}
	fieldRoots[0] = aggregationRoot[:]

	// Attestation data.
	attDataRoot, err := attestationDataRoot(att.Data)
	if err != nil {
		return [32]byte{}, nil
	}
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
		return [32]byte{}, nil
	}
	return root, nil
}

func validatorRoot(validator *ethpb.Validator) ([32]byte, error) {
	fieldRoots := make([][]byte, 8)

	// Public key.
	pubKeyChunks, err := pack([][]byte{validator.PublicKey})
	if err != nil {
		return [32]byte{}, nil
	}
	pubKeyRoot, err := bitwiseMerkleize(pubKeyChunks, uint64(len(pubKeyChunks)), uint64(len(pubKeyChunks)))
	if err != nil {
		return [32]byte{}, nil
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
		return [32]byte{}, nil
	}
	return root, nil
}

func eth1Root(eth1Data *ethpb.Eth1Data) ([32]byte, error) {
	fieldRoots := make([][]byte, 3)
	for i := 0; i < len(fieldRoots); i++ {
		fieldRoots[i] = make([]byte, 32)
	}
	if eth1Data != nil {
		if len(eth1Data.DepositRoot) > 0 {
			fieldRoots[0] = eth1Data.DepositRoot
		}
		eth1DataCountBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(eth1DataCountBuf, eth1Data.DepositCount)
		eth1CountRoot := bytesutil.ToBytes32(eth1DataCountBuf)
		fieldRoots[1] = eth1CountRoot[:]
		if len(eth1Data.BlockHash) > 0 {
			fieldRoots[2] = eth1Data.BlockHash
		}
	}
	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, nil
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

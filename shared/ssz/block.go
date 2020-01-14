package ssz

import (
	"encoding/binary"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

const BytesPerLengthOffset = 4

func beaconBlockSize(blk *ethpb.BeaconBlock) int {
	size := 0
	// Slot.
	size += 8
	// ParentRoot.
	size += 32
	// StateRoot.
	size += 32

	// BodySize.
	size += bodySize(blk.Body) + BytesPerLengthOffset
	return size
}

func bodySize(body *ethpb.BeaconBlockBody) int {
	size := 0
	// RandaoReveal.
	size += 96
	// Eth1Data.
	size += 32 + 8 + 32
	// Graffiti.
	size += 32

	// ProposerSlashings.
	size += BytesPerLengthOffset
	blockHeaderSize := 8 + 32 + 32 + 32
	for i := 0; i < len(body.ProposerSlashings); i++ {
		size += 8
		size += blockHeaderSize
		size += blockHeaderSize
	}

	// AttesterSlashings.
	size += BytesPerLengthOffset
	// Slot + index + block root + target checkpoint + source checkpoint.
	attDataSize := 8 + 8 + 32 + (8 + 32) + (8 + 32)
	for i := 0; i < len(body.AttesterSlashings); i++ {
		size += len(body.AttesterSlashings[i].Attestation_1.AttestingIndices) + BytesPerLengthOffset
		size += attDataSize
		size += 96
		size += len(body.AttesterSlashings[i].Attestation_2.AttestingIndices) + BytesPerLengthOffset
		size += attDataSize
		size += 96
	}

	// Attestations.
	return size
}

func MarshalBeaconBlock(blk *ethpb.BeaconBlock) []byte {
	fixedSizePosition := 0
	// We marshal the slot.
	slotBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBuf, blk.Slot)
	fixedSizePosition += 8

	// We consider the blk.ParentRoot as well.
	fixedSizePosition += 32
	// We consider blk.StateRoot as well.
	fixedSizePosition += 32

	// We start looking at the body.
	currentOffsetIndex := fixedSizePosition
	nextOffsetIndex := currentOffsetIndex

	return nil
}

func marshalBlockBody(body *ethpb.BeaconBlockBody, startOffset uint64) []byte {
	fixedSizePosition := 0
	// We consider the body.RandaoReveal
	fixedSizePosition += 32

	// Next we marshal the eth1 data.
	encEth1Data := marshalEth1Data(body.Eth1Data)
	fixedSizePosition += len(encEth1Data)

	// We consider the body.Graffiti
	fixedSizePosition += 32
}

func marshalEth1Data(data *ethpb.Eth1Data) []byte {
	res := make([]byte, 72)
	copy(res[0:32], data.DepositRoot)
	binary.LittleEndian.PutUint64(res[32:40], data.DepositCount)
	copy(res[40:72], data.BlockHash)
	return res
}

//func blockHeaderRoot(header *ethpb.BeaconBlockHeader) ([32]byte, error) {
//	fieldRoots := make([][]byte, 4)
//	if header != nil {
//		headerSlotBuf := make([]byte, 8)
//		binary.LittleEndian.PutUint64(headerSlotBuf, header.Slot)
//		headerSlotRoot := bytesutil.ToBytes32(headerSlotBuf)
//		fieldRoots[0] = headerSlotRoot[:]
//		fieldRoots[1] = header.ParentRoot
//		fieldRoots[2] = header.StateRoot
//		fieldRoots[3] = header.BodyRoot
//	}
//	return bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
//}
//
//func eth1Root(eth1Data *ethpb.Eth1Data) ([32]byte, error) {
//	fieldRoots := make([][]byte, 3)
//	for i := 0; i < len(fieldRoots); i++ {
//		fieldRoots[i] = make([]byte, 32)
//	}
//	if eth1Data != nil {
//		if len(eth1Data.DepositRoot) > 0 {
//			fieldRoots[0] = eth1Data.DepositRoot
//		}
//		eth1DataCountBuf := make([]byte, 8)
//		binary.LittleEndian.PutUint64(eth1DataCountBuf, eth1Data.DepositCount)
//		eth1CountRoot := bytesutil.ToBytes32(eth1DataCountBuf)
//		fieldRoots[1] = eth1CountRoot[:]
//		if len(eth1Data.BlockHash) > 0 {
//			fieldRoots[2] = eth1Data.BlockHash
//		}
//	}
//	return bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
//}
//
//func eth1DataVotesRoot(eth1DataVotes []*ethpb.Eth1Data) ([32]byte, error) {
//	eth1VotesRoots := make([][]byte, 0)
//	for i := 0; i < len(eth1DataVotes); i++ {
//		eth1, err := eth1Root(eth1DataVotes[i])
//		if err != nil {
//			return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
//		}
//		eth1VotesRoots = append(eth1VotesRoots, eth1[:])
//	}
//	eth1Chunks, err := pack(eth1VotesRoots)
//	if err != nil {
//		return [32]byte{}, errors.Wrap(err, "could not chunk eth1 votes roots")
//	}
//	eth1VotesRootsRoot, err := bitwiseMerkleize(eth1Chunks, uint64(len(eth1Chunks)), params.BeaconConfig().SlotsPerEth1VotingPeriod)
//	if err != nil {
//		return [32]byte{}, errors.Wrap(err, "could not compute eth1data votes merkleization")
//	}
//	eth1VotesRootBuf := new(bytes.Buffer)
//	if err := binary.Write(eth1VotesRootBuf, binary.LittleEndian, uint64(len(eth1DataVotes))); err != nil {
//		return [32]byte{}, errors.Wrap(err, "could not marshal eth1data votes length")
//	}
//	// We need to mix in the length of the slice.
//	eth1VotesRootBufRoot := make([]byte, 32)
//	copy(eth1VotesRootBufRoot, eth1VotesRootBuf.Bytes())
//	return mixInLength(eth1VotesRootsRoot, eth1VotesRootBufRoot), nil
//}

package ssz

import (
	"encoding/binary"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

const BytesPerLengthOffset = 4

// MarshalBeaconBlock --
func MarshalBeaconBlock(blk *ethpb.BeaconBlock) []byte {
	buf := make([]byte, beaconBlockSize(blk))
	fixedIndex := 0
	// We marshal the slot.
	binary.LittleEndian.PutUint64(buf[fixedIndex:fixedIndex+8], blk.Slot)
	fixedIndex += 8

	// We consider the blk.ParentRoot as well.
	copy(buf[fixedIndex:fixedIndex+32], blk.ParentRoot)
	fixedIndex += 32

	// We consider blk.StateRoot as well.
	copy(buf[fixedIndex:fixedIndex+32], blk.StateRoot)
	fixedIndex += 32

	// We marshal the block body. Given the body has variable sized elements, we
	// need to determine the start index for writing its offset in the encoded buffer.
	startOffsetIndex := 8 + 32 + 32 + BytesPerLengthOffset
	marshalBlockBody(blk.Body, buf, startOffsetIndex)
	offsetBuf := make([]byte, BytesPerLengthOffset)
	binary.LittleEndian.PutUint32(offsetBuf, uint32(startOffsetIndex))
	copy(buf[fixedIndex:fixedIndex+BytesPerLengthOffset], offsetBuf)
	return buf
}

func marshalBlockBody(body *ethpb.BeaconBlockBody, buf []byte, startOffset int) {
	fixedIndex := startOffset
	// RandaoReveal.
	copy(buf[fixedIndex:fixedIndex+96], body.RandaoReveal)
	fixedIndex += 96

	// Eth1Data.
	fixedIndex = marshalEth1Data(body.Eth1Data, buf, fixedIndex)

	// Graffiti..
	copy(buf[fixedIndex:fixedIndex+32], body.RandaoReveal)
	fixedIndex += 32
}

func marshalEth1Data(data *ethpb.Eth1Data, buf []byte, startOffset int) int {
	fixedIndex := startOffset
	copy(buf[fixedIndex:fixedIndex+32], data.DepositRoot)
	fixedIndex += 32
	binary.LittleEndian.PutUint64(buf[fixedIndex:fixedIndex+8], data.DepositCount)
	copy(buf[fixedIndex:fixedIndex+32], data.BlockHash)
	return fixedIndex
}

func beaconBlockSize(blk *ethpb.BeaconBlock) int {
	size := 0
	// Slot.
	size += 8
	// ParentRoot.
	size += 32
	// StateRoot.
	size += 32

	body := blk.Body
	size += BytesPerLengthOffset

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
		size += len(body.AttesterSlashings[i].Attestation_1.AttestingIndices)*8 + BytesPerLengthOffset
		size += attDataSize
		size += 96
		size += len(body.AttesterSlashings[i].Attestation_2.AttestingIndices)*8 + BytesPerLengthOffset
		size += attDataSize
		size += 96
	}

	// Attestations.
	size += BytesPerLengthOffset
	for i := 0; i < len(body.Attestations); i++ {
		size += int(body.Attestations[i].AggregationBits.Len()) + BytesPerLengthOffset
		size += attDataSize
		size += 96
	}

	// Deposits.
	size += BytesPerLengthOffset
	// Public key + withdrawal credentials + amount + signature.
	depositDataSize := 48 + 32 + 8 + 96
	treeDepth := 32
	for i := 0; i < len(body.Deposits); i++ {
		size += depositDataSize
		// (Deposit contract tree depth+1)*len(leaf).
		size += (treeDepth + 1) * 32
	}

	// VoluntaryExits.
	size += BytesPerLengthOffset
	for i := 0; i < len(body.VoluntaryExits); i++ {
		size += 8 + 8 + 96
	}
	return size
}

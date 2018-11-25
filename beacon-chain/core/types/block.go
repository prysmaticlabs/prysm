package types

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Block defines a beacon chain core primitive.
type Block struct {
	data *pb.BeaconBlock
}

// NewBlock explicitly sets the data field of a block.
// Return block with default fields if data is nil.
func NewBlock(data *pb.BeaconBlock) *Block {
	if data == nil {
		var ancestorHashes = make([][]byte, 0, 32)

		// It is assumed when data==nil, a genesis block will be returned.
		return &Block{
			data: &pb.BeaconBlock{
				AncestorHashes:        ancestorHashes,
				RandaoReveal:          []byte{0},
				PowChainRef:           []byte{0},
				ActiveStateRoot:       []byte{0},
				CrystallizedStateRoot: []byte{0},
				Specials:              []*pb.SpecialRecord{},
			},
		}
	}

	return &Block{data: data}
}

// SlotNumber of the beacon block.
func (b *Block) SlotNumber() uint64 {
	return b.data.Slot
}

// ParentHash corresponding to parent beacon block.
func (b *Block) ParentHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.AncestorHashes[0])
	return h
}

// Package types defines the essential types used throughout the beacon-chain.
package types

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
)

// BlockRegistry is a struct used to stor the blockhashes of blocks saved in the DB.
type BlockRegistry struct {
	data *pb.BlockRegistry
}

// NewRegistry creates a new block registry from the protobuf data.
func NewRegistry(data *pb.BlockRegistry) *BlockRegistry {

	if data != nil {
		return &BlockRegistry{data: data}
	}

	newdata := &pb.BlockRegistry{Blockhashes: make([][]byte, 0)}
	return &BlockRegistry{data: newdata}
}

// Proto returns the underlying protobuf data within the block registry struct.
func (b *BlockRegistry) Proto() *pb.BlockRegistry {
	return b.data
}

// Marshal encodes block registry object into the wire format.
func (b *BlockRegistry) Marshal() ([]byte, error) {
	return proto.Marshal(b.data)
}

// BlockHashes returns the hashes stored in the registry.
func (b *BlockRegistry) BlockHashes() [][]byte {
	return b.data.Blockhashes
}

// Block defines a beacon chain core primitive.
type Block struct {
	data *pb.BeaconBlock
}

// NewBlock explicitly sets the data field of a block.
// Return block with default fields if data is nil.
func NewBlock(data *pb.BeaconBlock) *Block {
	if data == nil {
		return &Block{
			data: &pb.BeaconBlock{
				ParentHash:            []byte{0},
				SlotNumber:            0,
				RandaoReveal:          []byte{0},
				Attestations:          []*pb.AttestationRecord{},
				PowChainRef:           []byte{0},
				ActiveStateHash:       []byte{0},
				CrystallizedStateHash: []byte{0},
				Timestamp:             ptypes.TimestampNow(),
			},
		}
	}

	return &Block{data: data}
}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
//
// TODO: Add more default fields.
func NewGenesisBlock() (*Block, error) {
	protoGenesis, err := ptypes.TimestampProto(time.Unix(0, 0))
	if err != nil {
		return nil, err
	}
	return &Block{
		data: &pb.BeaconBlock{
			Timestamp:  protoGenesis,
			ParentHash: []byte{},
		},
	}, nil
}

// Proto returns the underlying protobuf data within a block primitive.
func (b *Block) Proto() *pb.BeaconBlock {
	return b.data
}

// Marshal encodes block object into the wire format.
func (b *Block) Marshal() ([]byte, error) {
	return proto.Marshal(b.data)
}

// Hash generates the blake2b hash of the block
func (b *Block) Hash() ([32]byte, error) {
	data, err := proto.Marshal(b.data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal block proto data: %v", err)
	}
	var hash [32]byte
	h := blake2b.Sum512(data)
	copy(hash[:], h[:32])
	return hash, nil
}

// ParentHash corresponding to parent beacon block.
func (b *Block) ParentHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.ParentHash)
	return h
}

// SlotNumber of the beacon block.
func (b *Block) SlotNumber() uint64 {
	return b.data.SlotNumber
}

// PowChainRef returns a keccak256 hash corresponding to a PoW chain block.
func (b *Block) PowChainRef() common.Hash {
	return common.BytesToHash(b.data.PowChainRef)
}

// RandaoReveal returns the blake2b randao hash.
func (b *Block) RandaoReveal() [32]byte {
	var h [32]byte
	copy(h[:], b.data.RandaoReveal)
	return h
}

// ActiveStateHash returns the active state hash.
func (b *Block) ActiveStateHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.ActiveStateHash)
	return h
}

// CrystallizedStateHash returns the crystallized state hash.
func (b *Block) CrystallizedStateHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.CrystallizedStateHash)
	return h
}

// AttestationCount returns the number of attestations.
func (b *Block) AttestationCount() int {
	return len(b.data.Attestations)
}

// Attestations returns an array of attestations in the block.
func (b *Block) Attestations() []*pb.AttestationRecord {
	return b.data.Attestations
}

// Timestamp returns the Go type time.Time from the protobuf type contained in the block.
func (b *Block) Timestamp() (time.Time, error) {
	return ptypes.Timestamp(b.data.Timestamp)
}

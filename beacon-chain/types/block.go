package types

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"golang.org/x/crypto/blake2b"
)

// Block defines a beacon chain core primitive.
type Block struct {
	data *pb.BeaconBlockResponse
}

// AggregateVote contains the fields of aggregate vote in individual shard.
type AggregateVote struct {
	ShardID        uint32 // Shard ID of the voted shard.
	ShardBlockHash []byte // ShardBlockHash is the shard block hash of the voted shard.
	SignerBitmask  []byte // SignerBitmask is the bit mask of every validator that signed.
	AggregateSig   []uint // AggregateSig is the aggregated signatures of individual shard.
}

// NewBlock explicitly sets the data field of a block.
func NewBlock(data *pb.BeaconBlockResponse) (*Block, error) {
	if len(data.ParentHash) != 32 {
		return nil, errors.New("invalid block data, parent hash should be 32 bytes")
	}

	return &Block{data}, nil
}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock() (*Block, error) {
	genesisTime := time.Date(2018, time.July, 21, 12, 0, 0, 0, time.UTC)
	protoGenesis, err := ptypes.TimestampProto(genesisTime)
	if err != nil {
		return nil, err
	}
	// TODO: Add more default fields.
	return &Block{data: &pb.BeaconBlockResponse{Timestamp: protoGenesis}}, nil
}

// Proto returns the underlying protobuf data within a block primitive.
func (b *Block) Proto() *pb.BeaconBlockResponse {
	return b.data
}

// Hash generates the blake2b hash of the block
func (b *Block) Hash() ([32]byte, error) {
	data, err := proto.Marshal(b.data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal block proto data: %v", err)
	}
	return blake2b.Sum256(data), nil
}

// ParentHash corresponding to parent beacon block.
func (b *Block) ParentHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.ParentHash[:32])
	return h
}

// SlotNumber of the beacon block.
func (b *Block) SlotNumber() uint64 {
	return b.data.SlotNumber
}

// MainChainRef returns a keccak256 hash corresponding to a PoW chain block.
func (b *Block) MainChainRef() common.Hash {
	return common.BytesToHash(b.data.MainChainRef)
}

// RandaoReveal returns the blake2b randao hash.
func (b *Block) RandaoReveal() [32]byte {
	var h [32]byte
	copy(h[:], b.data.RandaoReveal[:32])
	return h
}

// ActiveStateHash blake2b value.
func (b *Block) ActiveStateHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.ActiveStateHash[:32])
	return h
}

// CrystallizedStateHash blake2b value.
func (b *Block) CrystallizedStateHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.CrystallizedStateHash[:32])
	return h
}

// Timestamp returns the Go type time.Time from the protobuf type contained in the block.
func (b *Block) Timestamp() (time.Time, error) {
	return ptypes.Timestamp(b.data.Timestamp)
}

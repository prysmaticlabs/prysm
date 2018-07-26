package types

import (
	"hash"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
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

// NewBlock creates a new beacon block given certain arguments.
func NewBlock(slotNumber uint64) *Block {
	data := &pb.BeaconBlockResponse{Timestamp: ptypes.TimestampNow(), SlotNumber: slotNumber}
	return &Block{data}
}

// NewBlockWithData explicitly sets the data field of a block.
func NewBlockWithData(data *pb.BeaconBlockResponse) *Block {
	return &Block{data}
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

// Hash generates the blake2b hash of the block
func (b *Block) Hash() (hash.Hash, error) {
	data, err := rlp.EncodeToBytes(b.data)
	if err != nil {
		return nil, err
	}
	return blake2b.New256(data)
}

// ParentHash corresponding to parent beacon block.
func (b *Block) ParentHash() (hash.Hash, error) {
	return blake2b.New256(b.data.ParentHash)
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
func (b *Block) RandaoReveal() (hash.Hash, error) {
	return blake2b.New256(b.data.MainChainRef)
}

// ActiveStateHash blake2b value.
func (b *Block) ActiveStateHash() (hash.Hash, error) {
	return blake2b.New256(b.data.ActiveStateHash)
}

// CrystallizedStateHash blake2b value.
func (b *Block) CrystallizedStateHash() (hash.Hash, error) {
	return blake2b.New256(b.data.CrystallizedStateHash)
}

// Timestamp returns the Go type time.Time from the protobuf type contained in the block.
func (b *Block) Timestamp() (time.Time, error) {
	return ptypes.Timestamp(b.data.Timestamp)
}

// InsertActiveHash updates the activeStateHash property in the data of a beacon block.
func (b *Block) InsertActiveHash(hash []byte) {
	b.data.ActiveStateHash = hash
}

// InsertCrystallizedHash updates the crystallizedStateHash property in the data of a beacon block.
func (b *Block) InsertCrystallizedHash(hash []byte) {
	b.data.CrystallizedStateHash = hash
}

package types

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Block defines a beacon chain core primitive.
type Block struct {
	data *Data
}

// Data getter makes the block's properties read-only.
func (b *Block) Data() *Data {
	return b.data
}

// NewBlock creates a new beacon block given certain arguments.
func NewBlock(slotNumber uint64) *Block {
	data := &Data{Timestamp: time.Now(), SlotNumber: slotNumber}
	return &Block{data}
}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock() *Block {
	timestamp := time.Date(2018, time.July, 21, 12, 0, 0, 0, time.UTC)
	// TODO: Add more default fields.
	return &Block{data: &Data{Timestamp: timestamp}}
}

// Data contains the fields in a beacon chain block.
type Data struct {
	ParentHash              common.Hash     // ParentHash is the hash of the parent beacon block.
	SlotNumber              uint64          // Slot number is the number a client should check to know when it creates block.
	RandaoReveal            common.Hash     // RandaoReveal is used for Randao commitment reveal.
	AttestationBitmask      []byte          // AttestationBitmask is the bit field of who from the attestation committee participated.
	AttestationAggregateSig []uint          // AttestationAggregateSig is validator's aggregate sig.
	ShardAggregateVotes     []AggregateVote // ShardAggregateVotes is shard aggregate votes.
	MainChainRef            common.Hash     // MainChainRef is the reference to main chain block.
	ActiveStateHash         []byte          // ActiveStateHash is the state that changes every block.
	CrystallizedStateHash   []byte          // CrystallizedStateHash is the state that changes every epoch.
	Timestamp               time.Time
}

// AggregateVote contains the fields of aggregate vote in individual shard.
type AggregateVote struct {
	ShardID        uint16      // Shard ID of the voted shard.
	ShardBlockHash common.Hash // ShardBlockHash is the shard block hash of the voted shard.
	SignerBitmask  []byte      // SignerBitmask is the bit mask of every validator that signed.
	AggregateSig   []uint      // AggregateSig is the aggregated signatures of individual shard.
}

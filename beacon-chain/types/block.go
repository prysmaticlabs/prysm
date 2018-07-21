package types

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Block defines a beacon chain core primitive.
type Block struct {
	header *Header
}

// Header getter makes the property read-only.
func (b *Block) Header() *Header {
	return b.header
}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock() *Block {
	// TODO: fetch from persistent storage, otherwise create a new one.
	timestamp, _ := time.Parse("Sat July 21 12:00:00 UTC 2018", "Sat July 21 12:00:00 UTC 2018")
	return &Block{header: &Header{Timestamp: timestamp}}
}

// Header contains the block header fields in beacon chain.
type Header struct {
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

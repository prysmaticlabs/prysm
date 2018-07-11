package types

import "github.com/ethereum/go-ethereum/common"

// Header contains the block header fields in beacon chain.
type Header struct {
	ParentHash              common.Hash     // ParentHash is the hash of the parent beacon block.
	SkipCount               uint64          // SkipCount is the number of skips, this is used for the full PoS mechanism.
	RandaoReveal            common.Hash     // RandaoReveal is used for Randao commitment reveal.
	AttestationBitmask      []byte          // AttestationBitmask is the bit field of who from the attestation committee participated.
	AttestationAggregateSig []uint          // AttestationAggregateSig is validator's aggregate sig.
	ShardAggregateVotes     []AggregateVote // ShardAggregateVotes is shard aggregate votes.
	MainChainRef            common.Hash     // MainChainRef is the reference to main chain block.
	StateHash               []byte          // StateHash is the concatenation of crystallized and active state.
	Sig                     []uint          // Sig is the signature of the proposer.
}

// AggregateVote contains the fields of aggregate vote in individual shard.
type AggregateVote struct {
	ShardID        uint16      // Shard ID of the voted shard.
	ShardBlockHash common.Hash // ShardBlockHash is the shard block hash of the voted shard.
	SignerBitmask  []byte      // SignerBitmask is the bit mask of every validator that signed.
	AggregateSig   []uint      // AggregateSig is the aggregated signatures of individual shard.
}

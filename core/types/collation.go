package types

import (
	"github.com/ethereum/go-ethereum/common"
)

type Collation struct {
	Header CollationHeader // Header metadata
	TxList []Transaction   // List of transaction in this collation
}

type CollationHeader struct {
	ShardID              uint64         // Shard ID of the shard
	ExpectedPeriodNumber uint64         // Expected number in which this collation expects to be included
	PeriodStartPrevhash  common.Hash    // Hash of the last block before the expected period starts
	ParentCollectionHash common.Hash    // Hash of the parent collation
	TxListRoot           common.Hash    // Root hash of the trie holding the transactions included in this collation
	Coinbase             common.Address // Address chose by the creater of the shard header
	PostStateRoot        common.Hash    // New state root of the shard after this collation
	ReceiptsRoot         common.Hash    // Root hash of the receipt trie

	// TODO: Add signature
}

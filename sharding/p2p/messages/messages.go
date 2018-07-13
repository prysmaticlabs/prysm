package messages

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// CollationBodyRequest defines a p2p message being sent over subscription feeds
// by notaries to other notaries or to proposers.
type CollationBodyRequest struct {
	ChunkRoot *common.Hash
	ShardID   *big.Int
	Period    *big.Int
	Proposer  *common.Address
	Signature [32]byte
}

// CollationBodyResponse defines the p2p message response sent back
// to the requesting peer.
type CollationBodyResponse struct {
	HeaderHash *common.Hash
	Body       []byte
}

// TransactionBroadcast defines the p2p message broadcast from simulators
// to the rest of the actors for transactions to be included in collation.
type TransactionBroadcast struct {
	Transaction *types.Transaction
}

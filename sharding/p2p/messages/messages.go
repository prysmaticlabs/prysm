package messages

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// CollationBodyRequest defines a p2p message being sent over subscription feeds
// by attesters to other attesters or to proposers.
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

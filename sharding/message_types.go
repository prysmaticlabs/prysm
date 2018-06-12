package sharding

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type CollationBodyRequest struct {
	ChunkRoot *common.Hash
	ShardID   *big.Int
	Period    *big.Int
	Proposer  *common.Address
}

type CollationBodyResponse struct {
	HeaderHash *common.Hash
	Body       []byte
}

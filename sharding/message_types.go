package sharding

import "github.com/ethereum/go-ethereum/common"

type CollationBodyRequest struct {
	HeaderHash *common.Hash
}

type CollationBodyResponse struct {
	HeaderHash *common.Hash
	Body       []byte
}

// 2018 Prysmatic Labs
// This file is part of the prysmaticlabs/geth-sharding library.
//
// This file overrides the default protocol transaction interface to prep it
// for sharding

package sharding

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync/atomic"
	//"github.com/ethereum/go-ethereum/common/hexutil"
	//"github.com/ethereum/go-ethereum/rlp"
)

// Transaction Base Sharding Struct
type Transaction struct {
	data txdata
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	// Code Payload
	Payload []byte `json:"input"    gencodec:"required"`
	// TODO:
	// add accesslist, chainid, shardid,

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}

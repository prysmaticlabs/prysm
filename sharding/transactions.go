// 2018 Prysmatic Labs
// This file is part of the prysmaticlabs/geth-sharding library.
//
// This file overrides the default protocol transaction interface to prep it
// for sharding

package sharding

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"sync/atomic"
	//"github.com/ethereum/go-ethereum/rlp"
)

// ShardingTransaction Base Struct
type ShardingTransaction struct {
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

	// Sharding specific fields
	// TODO: Figure out exact format of accesslist. array of arrays of addr + prefixes?
	AccessList []common.Address `json:"accessList" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}

type txdataMarshaling struct {
	AccountNonce hexutil.Uint64
	Price        *hexutil.Big
	GasLimit     hexutil.Uint64
	Amount       *hexutil.Big
	Payload      hexutil.Bytes
	ChainID      hexutil.Uint64
	ShardID      hexutil.Uint64
}

func NewShardingTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accessList []common.Address) *ShardingTransaction {
	return newShardingTransaction(nonce, &to, amount, gasLimit, gasPrice, data, accessList)
}

func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accessList []common.Address) *ShardingTransaction {
	return newShardingTransaction(nonce, nil, amount, gasLimit, gasPrice, data, accessList)
}

func newShardingTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accessList []common.Address) *ShardingTransaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		AccessList:   accessList,
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}

	return &ShardingTransaction{data: d}
}

// ChainID determines the chain the tx will go into (this is 1 on the mainnet)
func (tx *ShardingTransaction) ChainID() *big.Int {
	return big.NewInt(1)
}

// ShardID determines the shard a transaction belongs to
func (tx *ShardingTransaction) ShardID() *big.Int {
	// TODO: figure out how to determine ShardID. 1 for now
	return big.NewInt(1)
}

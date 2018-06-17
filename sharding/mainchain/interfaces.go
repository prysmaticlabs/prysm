package mainchain

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

// Signer defines an interface that can read from the Ethereum mainchain as well call
// read-only methods and functions from the Sharding Manager Contract.
type Signer interface {
	Sign(hash common.Hash) ([]byte, error)
}

// ContractManager specifies an interface that defines both read/write
// operations on a contract in the Ethereum mainchain.
type ContractManager interface {
	ContractCaller
	ContractTransactor
}

// ContractCaller defines an interface that can read from a contract on the
// Ethereum mainchain as well as call its read-only methods and functions.
type ContractCaller interface {
	SMCCaller() *contracts.SMCCaller
	GetShardCount() (int64, error)
}

// ContractTransactor defines an interface that can transact with a contract on the
// Ethereum mainchain as well as call its methods and functions.
type ContractTransactor interface {
	SMCTransactor() *contracts.SMCTransactor
	CreateTXOpts(value *big.Int) (*bind.TransactOpts, error)
}

// Reader defines an interface for a struct that can read mainchain information
// such as blocks, transactions, receipts, and more. Useful for testing.
type Reader interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

package sharding

import (
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	cli "gopkg.in/urfave/cli.v1"
)

// ShardingClient defines a service that provides full control and shared access of
// necessary components for a sharded Ethereum blockchain.
type ShardingClient interface {
	Start() error
	Close() error
	Context() *cli.Context
	CreateTXOpts(*big.Int) (*bind.TransactOpts, error)
	ChainReader() ethereum.ChainReader
	Account() *accounts.Account
	SMCCaller() *contracts.SMCCaller
	SMCTransactor() *contracts.SMCTransactor
	DepositFlagSet() bool
	DataDirFlag() string
}

// ShardingActor refers to either a notary, proposer, or observer.
type ShardingActor interface {
	Start() error
	Stop() error
}

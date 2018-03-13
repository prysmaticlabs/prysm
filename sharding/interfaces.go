package sharding

import (
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	cli "gopkg.in/urfave/cli.v1"
)

type shardingClient interface {
	Account() *accounts.Account
	ChainReader() ethereum.ChainReader
	PendingStateEventer() ethereum.PendingStateEventer
	SMCCaller() *contracts.SMCCaller
	Client() *ethclient.Client
	Context() *cli.Context
}

package types

import "github.com/urfave/cli"

var (
	// Web3ProviderFlag defines a flag for a mainchain RPC endpoint.
	Web3ProviderFlag = cli.StringFlag{
		Name:  "web3provider",
		Usage: "a mainchain web3 provider endpoint",
		Value: "localhost:8545",
	}
)

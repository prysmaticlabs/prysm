package utils

import (
	"math/big"

	shardparams "github.com/prysmaticlabs/prysm/sharding/client"
	"github.com/urfave/cli"
)

var (
	// DepositFlag defines whether a node will withdraw ETH from the user's account.
	DepositFlag = cli.BoolFlag{
		Name:  "deposit",
		Usage: "To become a notary in a sharding node, " + new(big.Int).Div(shardparams.DefaultConfig.NotaryDeposit, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)).String() + " ETH will be deposited into SMC",
	}
	// ActorFlag defines the role of the sharding client. Either proposer, notary, or simulator.
	ActorFlag = cli.StringFlag{
		Name:  "actor",
		Usage: `use the --actor notary or --actor proposer to start a notary or proposer service in the sharding node. If omitted, the sharding node registers an Observer service that simply observes the activity in the sharded network`,
	}
	// ShardIDFlag specifies which shard to listen to.
	ShardIDFlag = cli.IntFlag{
		Name:  "shardid",
		Usage: `use the --shardid to determine which shard to start p2p server, listen for incoming transactions and perform proposer/observer duties`,
	}
)

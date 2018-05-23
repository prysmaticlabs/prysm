package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/node"
	"github.com/ethereum/go-ethereum/sharding/notary"
	"github.com/ethereum/go-ethereum/sharding/observer"
	"github.com/ethereum/go-ethereum/sharding/proposer"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	shardingCommand = cli.Command{
		Action:    utils.MigrateFlags(shardingCmd),
		Name:      "sharding",
		Usage:     "Start a sharding-enabled node",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{utils.ActorFlag, utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag, utils.DepositFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding node that manages services related to submitting collations to a Sharding Manager Contract, notary and proposer services, and shardp2p connections. This feature is a work in progress.
`,
	}
)

// shardingCmd is the main cmd line entry point for starting a sharding-enabled node.
// A sharding node launches a suite of services including notary services,
// proposer services, and a shardp2p protocol.
func shardingCmd(ctx *cli.Context) error {
	// configures a sharding-enabled node using the cli's context.
	shardingNode, err := node.NewNode(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize node instance: %v", err)
	}
	if err := registerShardingServices(shardingNode); err != nil {
		return fmt.Errorf("could not start sharding node: %v", err)
	}
	defer shardingNode.Close()
	// starts a connection to a geth node and kicks off every registered service.
	return shardingNode.Start()
}

// registerShardingServices sets up either a notary or proposer
// sharding service dependent on the ClientType cli flag. We should be defining
// the services we want to register here, as this is the geth command entry point
// for sharding.
func registerShardingServices(n node.Node) error {
	actorFlag := n.Context().GlobalString(utils.ActorFlag.Name)

	err := n.Register(func() (sharding.Service, error) {
		if actorFlag == "notary" {
			return notary.NewNotary(n)
		} else if actorFlag == "proposer" {
			return proposer.NewProposer(n)
		}
		return observer.NewObserver(n)
	})

	if err != nil {
		return fmt.Errorf("failed to register the main sharding services: %v", err)
	}

	// TODO(prestonvanloon) registers the shardp2p service.
	// we can do n.Register and initialize a shardp2p.NewServer() or something like that.
	return nil
}

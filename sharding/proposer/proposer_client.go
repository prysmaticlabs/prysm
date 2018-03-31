package proposer

import (
	"github.com/ethereum/go-ethereum/sharding/client"
	"github.com/ethereum/go-ethereum/log"
	cli "gopkg.in/urfave/cli.v1"
)

func NewProposerClient(ctx *cli.Context) *client.ShardingClient {
	c := client.MakeClient(ctx)
	return c

}

func ProposerStart(sclient *client.ShardingClient) error {
	log.Info("Starting proposer client")
	sclient.Start()

	return nil

}

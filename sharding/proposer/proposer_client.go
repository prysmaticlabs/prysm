package proposer

import (
	"github.com/ethereum/go-ethereum/sharding/client"
	cli "gopkg.in/urfave/cli.v1"
)

func NewProposerClient(ctx *cli.Context) *client.ShardingClient {
	c := client.MakeClient(ctx)
	return c

}

func ProposerStart(sclient *client.ShardingClient) error {
	sclient.Start()

	return nil

}

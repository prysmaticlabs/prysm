package collator

import (
	"github.com/ethereum/go-ethereum/sharding/client"
	cli "gopkg.in/urfave/cli.v1"
)

func NewCollatorClient(ctx *cli.Context) *client.ShardingClient {
	c := client.MakeClient(ctx)
	return c

}

func CollatorStart(sclient *client.ShardingClient) error {
	sclient.Start()

	if err := joinCollatorPool(sclient); err != nil {
		return err
	}

	if err := subscribeBlockHeaders(sclient); err != nil {
		return err
	}

	return nil

}

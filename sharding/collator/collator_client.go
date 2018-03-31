package collator

import (
	"github.com/ethereum/go-ethereum/sharding/client"
	"github.com/ethereum/go-ethereum/log"
	cli "gopkg.in/urfave/cli.v1"
)

func NewCollatorClient(ctx *cli.Context) *client.ShardingClient {
	c := client.MakeClient(ctx)
	return c

}

func CollatorStart(sclient *client.ShardingClient) error {
	log.Info("Starting collator client")
	rpcClient, err := sclient.Start()
	defer rpcClient.Close()
	if err != nil {
		return err
	}

	if err := joinCollatorPool(sclient); err != nil {
		return err
	}

	if err := subscribeBlockHeaders(sclient); err != nil {
		return err
	}

	return nil

}

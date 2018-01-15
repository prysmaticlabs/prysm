package sharding

import (
	"github.com/ethereum/go-ethereum/log"
	cli "gopkg.in/urfave/cli.v1"
)

type Client struct {
}

func MakeShardingClient(ctx *cli.Context) *Client {
	// TODO: Setup client
	return &Client{}
}

func (c *Client) Start() error {
	log.Info("Starting sharding client")
	// TODO: Dial to RPC
	// TODO: Verify VMC
	if err := c.verifyVMC(); err != nil {
		return err
	}

	return nil
}

func (c *Client) Wait() {
	// TODO: Blocking lock
}

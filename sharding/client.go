package sharding

import (
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	// TODO: Can this be referenced from main.clientIdentifier?
	clientIdentifier = "geth" // Client identifier to advertise over the network
)

type Client struct {
	endpoint string
	client   *rpc.Client
}

func MakeShardingClient(ctx *cli.Context) *Client {
	endpoint := ""
	if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		endpoint = ctx.GlobalString(utils.DataDirFlag.Name)
	}

	return &Client{
		endpoint: endpoint,
	}
}

func (c *Client) Start() error {
	log.Info("Starting sharding client")
	rpcClient, err := dialRPC(c.endpoint)
	if err != nil {
		return err
	}
	c.client = rpcClient
	defer c.client.Close()
	if err := c.verifyVMC(); err != nil {
		return err
	}

	// TODO: Wait to be selected?

	return nil
}

func (c *Client) Wait() {
	// TODO: Blocking lock
}

func dialRPC(endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	}
	return rpc.Dial(endpoint)
}

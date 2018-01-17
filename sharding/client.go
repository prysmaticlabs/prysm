package sharding

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethclient"
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
	client   *ethclient.Client
	keystore *keystore.KeyStore // Keystore containing the single signer
}

func MakeShardingClient(ctx *cli.Context) *Client {
	path := node.DefaultDataDir()
	if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		path = ctx.GlobalString(utils.DataDirFlag.Name)
	}
	endpoint := fmt.Sprintf("%s/geth.ipc", path)

	config := &node.Config{
		DataDir: "/tmp/ethereum",
	}
	scryptN, scryptP, keydir, err := config.AccountConfig()
	if err != nil {
		panic(err) // TODO: handle this
	}

	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	return &Client{
		endpoint: endpoint,
		keystore: ks,
	}
}

func (c *Client) Start() error {
	log.Info("Starting sharding client")
	rpcClient, err := dialRPC(c.endpoint)
	if err != nil {
		return err
	}
	c.client = ethclient.NewClient(rpcClient)
	defer rpcClient.Close()
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

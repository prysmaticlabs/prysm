package sharding

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "geth" // Used to determine the ipc name.
)

// Client for sharding. Communicates to geth node via JSON RPC.
type Client struct {
	endpoint string             // Endpoint to JSON RPC
	client   *ethclient.Client  // Ethereum RPC client.
	keystore *keystore.KeyStore // Keystore containing the single signer
	ctx      *cli.Context       // Command line context
	vmc      *contracts.VMC     // The deployed validator management contract
}

// MakeShardingClient for interfacing with geth full node.
func MakeShardingClient(ctx *cli.Context) *Client {
	path := node.DefaultDataDir()
	if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		path = ctx.GlobalString(utils.DataDirFlag.Name)
	}

	endpoint := ctx.Args().First()
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s/%s.ipc", path, clientIdentifier)
	}

	config := &node.Config{
		DataDir: path,
	}
	scryptN, scryptP, keydir, err := config.AccountConfig()
	if err != nil {
		panic(err) // TODO(prestonvanloon): handle this
	}
	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	return &Client{
		endpoint: endpoint,
		keystore: ks,
		ctx:      ctx,
	}
}

// Start the sharding client.
// * Connects to node.
// * Verifies or deploys the validator management contract.
func (c *Client) Start() error {
	log.Info("Starting sharding client")
	rpcClient, err := dialRPC(c.endpoint)
	if err != nil {
		return err
	}
	c.client = ethclient.NewClient(rpcClient)
	defer rpcClient.Close()
	if err := initVMC(c); err != nil {
		return err
	}

	// TODO: Wait to be selected as a collator, then start listening to the
	// geth node's txpool.

	// Listens to incoming transactions from the geth node's txpool and directs
	// them into the VMC if node is a collator
	if err := listenTXPool(c); err != nil {
		return err
	}
	return nil
}

// Wait until sharding client is shutdown.
func (c *Client) Wait() {
	// TODO: Blocking lock.
}

// dialRPC endpoint to node.
func dialRPC(endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	}
	return rpc.Dial(endpoint)
}

// UnlockAccount will unlock the specified account using utils.PasswordFileFlag or empty string if unset.
func (c *Client) unlockAccount(account accounts.Account) error {
	pass := ""

	if c.ctx.GlobalIsSet(utils.PasswordFileFlag.Name) {
		blob, err := ioutil.ReadFile(c.ctx.GlobalString(utils.PasswordFileFlag.Name))
		if err != nil {
			return fmt.Errorf("unable to read account password contents in file %s. %v", utils.PasswordFileFlag.Value, err)
		}
		// TODO: Use bufio.Scanner or other reader that doesn't include a trailing newline character.
		pass = strings.Trim(string(blob), "\n") // Some text files end in new line, remove with strings.Trim.
	}

	return c.keystore.Unlock(account, pass)
}

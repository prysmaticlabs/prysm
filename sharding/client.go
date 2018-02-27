package sharding

//go:generate abigen --sol contracts/validator_manager.sol --pkg contracts --out contracts/validator_manager.go

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	if ctx.GlobalIsSet(utils.IPCPathFlag.Name) {
		endpoint = ctx.GlobalString(utils.IPCPathFlag.Name)
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

	// Deposit 100ETH into the validator set in the VMC. Checks if account
	// is already a validator in the VMC (in the case the client restarted).
	// Once that's done we can subscribe to block headers.
	//
	// TODO: this function should store the validator's VMC index as a property
	// in the client's struct
	if err := joinValidatorSet(c); err != nil {
		return err
	}

	// Listens to block headers from the geth node and if we are an eligible
	// proposer, we fetch pending transactions and propose a collation
	if err := subscribeBlockHeaders(c); err != nil {
		return err
	}
	return nil
}

// Wait until sharding client is shutdown.
func (c *Client) Wait() {
	log.Info("Sharding client has been shutdown...")
}

// WatchCollationHeaders checks the logs for add_header func calls
// and updates the head collation of the client. We can probably store
// this as a property of the client struct
func (c *Client) WatchCollationHeaders() {

}

// UnlockAccount will unlock the specified account using utils.PasswordFileFlag or empty string if unset.
func (c *Client) unlockAccount(account accounts.Account) error {
	pass := ""

	if c.ctx.GlobalIsSet(utils.PasswordFileFlag.Name) {
		file, err := os.Open(c.ctx.GlobalString(utils.PasswordFileFlag.Name))
		if err != nil {
			return fmt.Errorf("unable to open file containing account password %s. %v", utils.PasswordFileFlag.Value, err)
		}
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanWords)
		if !scanner.Scan() {
			err = scanner.Err()
			if err != nil {
				return fmt.Errorf("unable to read contents of file %v", err)
			}
			return errors.New("password not found in file")
		}

		pass = scanner.Text()
	}

	return c.keystore.Unlock(account, pass)
}

func (c *Client) createTXOps(value *big.Int) (*bind.TransactOpts, error) {
	account, err := c.Account()
	if err != nil {
		return nil, err
	}

	return &bind.TransactOpts{
		From:  account.Address,
		Value: value,
		Signer: func(signer types.Signer, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			networkID, err := c.client.NetworkID(context.Background())
			if err != nil {
				return nil, fmt.Errorf("unable to fetch networkID: %v", err)
			}
			return c.keystore.SignTx(*account, tx, networkID /* chainID */)
		},
	}, nil
}

// Account to use for sharding transactions.
func (c *Client) Account() (*accounts.Account, error) {
	accounts := c.keystore.Accounts()
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts found")
	}

	if err := c.unlockAccount(accounts[0]); err != nil {
		return nil, fmt.Errorf("cannot unlock account. %v", err)
	}

	return &accounts[0], nil
}

// ChainReader for interacting with the chain.
func (c *Client) ChainReader() ethereum.ChainReader {
	return ethereum.ChainReader(c.client)
}

// Client to interact with ethereum node.
func (c *Client) Client() *ethclient.Client {
	return c.client
}

// VMCCaller to interact with the validator management contract.
func (c *Client) VMCCaller() *contracts.VMCCaller {
	return &c.vmc.VMCCaller
}

// dialRPC endpoint to node.
func dialRPC(endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	}
	return rpc.Dial(endpoint)
}

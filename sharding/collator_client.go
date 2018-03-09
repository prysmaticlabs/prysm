package sharding

//go:generate abigen --sol contracts/sharding_manager.sol --pkg contracts --out contracts/sharding_manager.go

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

// Client for Collator. Communicates to Geth node via JSON RPC.
type collatorClient struct {
	endpoint string             // Endpoint to JSON RPC
	client   *ethclient.Client  // Ethereum RPC client.
	keystore *keystore.KeyStore // Keystore containing the single signer
	ctx      *cli.Context       // Command line context
	smc      *contracts.SMC     // The deployed sharding management contract
}

// MakeCollatorClient for interfacing with Geth full node.
func MakeCollatorClient(ctx *cli.Context) *collatorClient {
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

	return &collatorClient{
		endpoint: endpoint,
		keystore: ks,
		ctx:      ctx,
	}
}

// Start the collator client.
// * Connects to Geth node.
// * Verifies or deploys the sharding manager contract.
func (c *collatorClient) Start() error {
	log.Info("Starting collator client")
	rpcClient, err := dialRPC(c.endpoint)
	if err != nil {
		return err
	}
	c.client = ethclient.NewClient(rpcClient)
	defer rpcClient.Close()

	// Check account existence and unlock account before starting collator client
	accounts := c.keystore.Accounts()
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	if err := c.unlockAccount(accounts[0]); err != nil {
		return fmt.Errorf("cannot unlock account. %v", err)
	}

	if err := initSMC(c); err != nil {
		return err
	}

	// Deposit 100ETH into the collator set in the SMC. Checks if account
	// is already a collator in the SMC (in the case the client restarted).
	// Once that's done we can subscribe to block headers.
	//
	// TODO: this function should store the collator's SMC index as a property
	// in the client's struct
	if err := joinCollatorPool(c); err != nil {
		return err
	}

	// Listens to block headers from the Geth node and if we are an eligible
	// collator, we fetch pending transactions and collator a collation
	if err := subscribeBlockHeaders(c); err != nil {
		return err
	}
	return nil
}

// Wait until collator client is shutdown.
func (c *collatorClient) Wait() {
	log.Info("Sharding client has been shutdown...")
}

// WatchCollationHeaders checks the logs for add_header func calls
// and updates the head collation of the client. We can probably store
// this as a property of the client struct
func (c *collatorClient) WatchCollationHeaders() {

}

// UnlockAccount will unlock the specified account using utils.PasswordFileFlag or empty string if unset.
func (c *collatorClient) unlockAccount(account accounts.Account) error {
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

func (c *collatorClient) createTXOps(value *big.Int) (*bind.TransactOpts, error) {
	account := c.Account()

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
func (c *collatorClient) Account() *accounts.Account {
	accounts := c.keystore.Accounts()

	return &accounts[0]
}

// ChainReader for interacting with the chain.
func (c *collatorClient) ChainReader() ethereum.ChainReader {
	return ethereum.ChainReader(c.client)
}

// Client to interact with ethereum node.
func (c *collatorClient) Client() *ethclient.Client {
	return c.client
}

// SMCCaller to interact with the sharding manager contract.
func (c *collatorClient) SMCCaller() *contracts.SMCCaller {
	return &c.smc.SMCCaller
}

// dialRPC endpoint to node.
func dialRPC(endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	}
	return rpc.Dial(endpoint)
}

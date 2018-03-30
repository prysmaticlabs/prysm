package sharding

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

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

// ShardingClient abstraction for collators and proposers
type ShardingClient struct {
	endpoint string             // Endpoint to JSON RPC
	client   *ethclient.Client  // Ethereum RPC client.
	keystore *keystore.KeyStore // Keystore containing the single signer
	ctx      *cli.Context       // Command line context
	smc      *contracts.SMC     // The deployed sharding management contract
}

// MakeShardingClient for interfacing with Geth full node.
func MakeShardingClient(ctx *cli.Context) *ShardingClient {
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

	return &ShardingClient{
		endpoint: endpoint,
		keystore: ks,
		ctx:      ctx,
	}
}

// UnlockAccount will unlock the specified account using utils.PasswordFileFlag or empty string if unset.
func (c *ShardingClient) UnlockAccount(account accounts.Account) error {
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

// CreateTXOps initializes a SMC call tx op
func (c *ShardingClient) CreateTXOps(value *big.Int) (*bind.TransactOpts, error) {
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
func (c *ShardingClient) Account() *accounts.Account {
	accounts := c.keystore.Accounts()

	return &accounts[0]
}

// ChainReader for interacting with the chain.
func (c *ShardingClient) ChainReader() ethereum.ChainReader {
	return ethereum.ChainReader(c.client)
}

// Client to interact with ethereum node.
func (c *ShardingClient) Client() *ethclient.Client {
	return c.client
}

// AttachClient attaches the ethclient RPC connection
func (c *ShardingClient) AttachClient(ec *ethclient.Client) *ethclient.Client {
	c.client = ec
	return c.client
}

// Context fetches the CLI context
func (c *ShardingClient) Context() *cli.Context {
	return c.ctx
}

// SMCCaller to interact with the sharding manager contract.
func (c *ShardingClient) SMCCaller() *contracts.SMCCaller {
	return &c.smc.SMCCaller
}

// SMCTransactor to interact with the sharding manager contract.
func (c *ShardingClient) SMCTransactor() *contracts.SMCTransactor {
	return &c.smc.SMCTransactor
}

// Endpoint fetches the rpc endpoint
func (c *ShardingClient) Endpoint() string {
	return c.endpoint
}

// InitSMC initializes the sharding manager contract bindings.
// If the SMC does not exist, it will be deployed.
func (c *ShardingClient) InitSMC() error {
	b, err := c.client.CodeAt(context.Background(), shardingManagerAddress, nil)
	if err != nil {
		return fmt.Errorf("unable to get contract code at %s: %v", shardingManagerAddress, err)
	}

	if len(b) == 0 {
		log.Info(fmt.Sprintf("No sharding manager contract found at %s. Deploying new contract.", shardingManagerAddress.String()))

		txOps, err := c.CreateTXOps(big.NewInt(0))
		if err != nil {
			return fmt.Errorf("unable to intiate the transaction: %v", err)
		}

		addr, tx, contract, err := contracts.DeploySMC(txOps, c.client)
		if err != nil {
			return fmt.Errorf("unable to deploy sharding manager contract: %v", err)
		}

		for pending := true; pending; _, pending, err = c.client.TransactionByHash(context.Background(), tx.Hash()) {
			if err != nil {
				return fmt.Errorf("unable to get transaction by hash: %v", err)
			}
			time.Sleep(1 * time.Second)
		}

		c.smc = contract
		log.Info(fmt.Sprintf("New contract deployed at %s", addr.String()))
	} else {
		contract, err := contracts.NewSMC(shardingManagerAddress, c.client)
		if err != nil {
			return fmt.Errorf("failed to create sharding contract: %v", err)
		}
		c.smc = contract
	}

	return nil
}

// dialRPC endpoint to node.
func dialRPC(endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	}
	return rpc.Dial(endpoint)
}

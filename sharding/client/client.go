// Package client provides an interface for interacting with a running ethereum full node.
// As part of the initial phases of sharding, actors will need access to the sharding management
// contract on the main PoW chain.
package client

import (
	"context"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "geth" // Used to determine the ipc name.
)

// General client for a sharding-enabled system.
// Communicates to Geth node via JSON RPC.
type shardingClient struct {
	endpoint  string             // Endpoint to JSON RPC.
	client    *ethclient.Client  // Ethereum RPC client.
	keystore  *keystore.KeyStore // Keystore containing the single signer.
	ctx       *cli.Context       // Command line context.
	smc       *contracts.SMC     // The deployed sharding management contract.
	rpcClient *rpc.Client        // The RPC client connection to the main geth node.
}

// Client methods that must be implemented to run a sharding node.
type Client interface {
	Start() error
	Close()
	CreateTXOpts(*big.Int) (*bind.TransactOpts, error)
	ChainReader() ethereum.ChainReader
	Account() *accounts.Account
	SMCCaller() *contracts.SMCCaller
	SMCTransactor() *contracts.SMCTransactor
	DepositFlagSet() bool
}

// NewClient setups the sharding config, registers the services required
// by the sharded system.
func NewClient(ctx *cli.Context) Client {
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
		panic(err) // TODO(prestonvanloon): handle this.
	}
	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	// Registers required services. Notary/Proposer are services in a sharding client,
	// and they are selected based on command line flags at runtime.

	c := &shardingClient{
		endpoint: endpoint,
		keystore: ks,
		ctx:      ctx,
	}
	if err := c.registerShardingServices(); err != nil {
		panic(err) // TODO(rauljordan): handle this.
	}
	return c
}

func (c *shardingClient) registerShardingServices() error {
	return nil
}

// Start the sharding client.
// Connects to Geth node.
// Verifies or deploys the sharding manager contract.
func (c *shardingClient) Start() error {
	rpcClient, err := dialRPC(c.endpoint)
	if err != nil {
		return fmt.Errorf("cannot start rpc client. %v", err)
	}
	c.rpcClient = rpcClient
	c.client = ethclient.NewClient(rpcClient)

	// Check account existence and unlock account before starting notary client.
	accounts := c.keystore.Accounts()
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	if err := unlockAccount(c, accounts[0]); err != nil {
		return fmt.Errorf("cannot unlock account. %v", err)
	}

	smc, err := initSMC(c)
	if err != nil {
		return err
	}
	c.smc = smc

	return nil
}

// Close the RPC client connection.
func (c *shardingClient) Close() {
	c.rpcClient.Close()
}

// CreateTXOpts creates a *TransactOpts with a signer using the default account on the keystore.
func (c *shardingClient) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
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
func (c *shardingClient) Account() *accounts.Account {
	accounts := c.keystore.Accounts()

	return &accounts[0]
}

// ChainReader for interacting with the chain.
func (c *shardingClient) ChainReader() ethereum.ChainReader {
	return ethereum.ChainReader(c.client)
}

// Client to interact with ethereum node.
func (c *shardingClient) ethereumClient() *ethclient.Client {
	return c.client
}

// SMCCaller to interact with the sharding manager contract.
func (c *shardingClient) SMCCaller() *contracts.SMCCaller {
	return &c.smc.SMCCaller
}

// SMCTransactor allows us to send tx's to the SMC programmatically.
func (c *shardingClient) SMCTransactor() *contracts.SMCTransactor {
	return &c.smc.SMCTransactor
}

// DepositFlagSet returns true for cli flag --deposit.
func (c *shardingClient) DepositFlagSet() bool {
	return c.ctx.GlobalBool(utils.DepositFlag.Name)
}

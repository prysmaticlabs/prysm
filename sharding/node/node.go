// Package node provides an interface for interacting with a running ethereum full node.
// As part of the initial phases of sharding, actors will need access to the sharding management
// contract on the main PoW chain.
package node

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync"

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
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "geth" // Used to determine the ipc name.
)

// Node methods that must be implemented to run a sharding node.
type Node interface {
	Start() error
	Close()
	Context() *cli.Context
	Register(sharding.ServiceConstructor) error
	CreateTXOpts(*big.Int) (*bind.TransactOpts, error)
	ChainReader() ethereum.ChainReader
	Account() *accounts.Account
	SMCCaller() *contracts.SMCCaller
	SMCFilterer() *contracts.SMCFilterer
	SMCTransactor() *contracts.SMCTransactor
	DepositFlagSet() bool
}

// General node for a sharding-enabled system.
// Communicates to Geth node via JSON RPC.
type shardingNode struct {
	endpoint     string                        // Endpoint to JSON RPC.
	client       *ethclient.Client             // Ethereum RPC client.
	keystore     *keystore.KeyStore            // Keystore containing the single signer.
	ctx          *cli.Context                  // Command line context.
	smc          *contracts.SMC                // The deployed sharding management contract.
	rpcClient    *rpc.Client                   // The RPC client connection to the main geth node.
	lock         sync.Mutex                    // Mutex lock for concurrency management.
	serviceFuncs []sharding.ServiceConstructor // Stores an array of service callbacks to start upon running.
}

// NewNode setups the sharding config, registers the services required
// by the sharded system.
func NewNode(ctx *cli.Context) (Node, error) {
	c := &shardingNode{ctx: ctx}
	// Sets up all configuration options based on cli flags.
	if err := c.configShardingNode(); err != nil {
		return nil, err
	}
	return c, nil
}

// Start is the main entrypoint of a sharding node. It starts off every service
// attached to it.
func (n *shardingNode) Start() error {
	// Sets up a connection to a Geth node via RPC.
	rpcClient, err := dialRPC(n.endpoint)
	if err != nil {
		return fmt.Errorf("cannot start rpc client. %v", err)
	}
	n.rpcClient = rpcClient
	n.client = ethclient.NewClient(rpcClient)

	// Check account existence and unlock account before starting.
	accounts := n.keystore.Accounts()
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	if err := n.unlockAccount(accounts[0]); err != nil {
		return fmt.Errorf("cannot unlock account. %v", err)
	}

	// Initializes bindings to SMC.
	smc, err := initSMC(n)
	if err != nil {
		return err
	}
	n.smc = smc

	// Starts every service attached to the sharding node.
	for _, serviceFunc := range n.serviceFuncs {
		// Initializes each service.
		service, err := serviceFunc()
		if err != nil {
			return err
		}
		if err := service.Start(); err != nil {
			// Handles the stopping of a service on error.
			service.Stop()
			return err
		}
	}
	return nil
}

// configShardingNode uses cli flags to configure the data
// directory, ipc endpoints, keystores, and more.
func (n *shardingNode) configShardingNode() error {
	path := node.DefaultDataDir()
	if n.ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		path = n.ctx.GlobalString(utils.DataDirFlag.Name)
	}

	endpoint := n.ctx.Args().First()
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s/%s.ipc", path, clientIdentifier)
	}
	if n.ctx.GlobalIsSet(utils.IPCPathFlag.Name) {
		endpoint = n.ctx.GlobalString(utils.IPCPathFlag.Name)
	}

	config := &node.Config{
		DataDir: path,
	}

	scryptN, scryptP, keydir, err := config.AccountConfig()
	if err != nil {
		return err
	}

	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	n.endpoint = endpoint
	n.keystore = ks

	return nil
}

// Register appends a struct to the sharding node's services that
// satisfies the Service interface containing lifecycle handlers. Notary, Proposer,
// and ShardP2P are examples of services. The rationale behind this is that the
// sharding node should know very little about the functioning of its underlying
// services as they should be extensible.
func (n *shardingNode) Register(constructor sharding.ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

// Close the RPC client connection.
func (n *shardingNode) Close() {
	n.rpcClient.Close()
}

// CreateTXOpts creates a *TransactOpts with a signer using the default account on the keystore.
func (n *shardingNode) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	account := n.Account()

	return &bind.TransactOpts{
		From:  account.Address,
		Value: value,
		Signer: func(signer types.Signer, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			networkID, err := n.client.NetworkID(context.Background())
			if err != nil {
				return nil, fmt.Errorf("unable to fetch networkID: %v", err)
			}
			return n.keystore.SignTx(*account, tx, networkID /* chainID */)
		},
	}, nil
}

// Account to use for sharding transactions.
func (n *shardingNode) Account() *accounts.Account {
	accounts := n.keystore.Accounts()

	return &accounts[0]
}

// Context returns the cli context.
func (n *shardingNode) Context() *cli.Context {
	return n.ctx
}

// ChainReader for interacting with the chain.
func (n *shardingNode) ChainReader() ethereum.ChainReader {
	return ethereum.ChainReader(n.client)
}

// SMCCaller to interact with the sharding manager contract.
func (n *shardingNode) SMCCaller() *contracts.SMCCaller {
	return &n.smc.SMCCaller
}

// SMCTransactor allows us to send tx's to the SMC programmatically.
func (n *shardingNode) SMCTransactor() *contracts.SMCTransactor {
	return &n.smc.SMCTransactor
}

// SMCFilterer allows us to watch events generated from the SMC
func (n *shardingNode) SMCFilterer() *contracts.SMCFilterer {
	return &n.smc.SMCFilterer
}

// DepositFlagSet returns true for cli flag --deposit.
func (n *shardingNode) DepositFlagSet() bool {
	return n.ctx.GlobalBool(utils.DepositFlag.Name)
}

// Client to interact with a geth node via JSON-RPC.
func (n *shardingNode) ethereumClient() *ethclient.Client {
	return n.client
}

// unlockAccount will unlock the specified account using utils.PasswordFileFlag
// or empty string if unset.
func (n *shardingNode) unlockAccount(account accounts.Account) error {
	pass := ""

	if n.ctx.GlobalIsSet(utils.PasswordFileFlag.Name) {
		file, err := os.Open(n.ctx.GlobalString(utils.PasswordFileFlag.Name))
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

	return n.keystore.Unlock(account, pass)
}

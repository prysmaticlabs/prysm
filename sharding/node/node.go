// Package sharding/node provides an interface for interacting with a running ethereum full node.
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
	"github.com/ethereum/go-ethereum/sharding/notary"
	"github.com/ethereum/go-ethereum/sharding/proposer"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "geth" // Used to determine the ipc name.
)

// General node for a sharding-enabled system.
// Communicates to Geth node via JSON RPC.
type shardingNode struct {
	endpoint  string             // Endpoint to JSON RPC.
	client    *ethclient.Client  // Ethereum RPC client.
	keystore  *keystore.KeyStore // Keystore containing the single signer.
	ctx       *cli.Context       // Command line context.
	smc       *contracts.SMC     // The deployed sharding management contract.
	rpcClient *rpc.Client        // The RPC client connection to the main geth node.
}

// Node methods that must be implemented to run a sharding node.
type Node interface {
	Start() error
	Close()
	CreateTXOpts(*big.Int) (*bind.TransactOpts, error)
	ChainReader() ethereum.ChainReader
	Account() *accounts.Account
	SMCCaller() *contracts.SMCCaller
	SMCTransactor() *contracts.SMCTransactor
	DepositFlagSet() bool
}

// NewCNode setups the sharding config, registers the services required
// by the sharded system.
func NewNode(ctx *cli.Context) Client {
	c := &shardingNode{ctx: ctx}

	// Sets up all configuration options based on cli flags.
	if err := c.configShardingNode(); err != nil {
		panic(err) // TODO(rauljordan): handle this
	}

	// Registers all required services the sharding node will run upon start.
	// These include shardp2p servers, notary/proposer event loops, and more.
	if err := c.registerShardingServices(); err != nil {
		panic(err) // TODO(rauljordan): handle this.
	}
	return c
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
		panic(err) // TODO(prestonvanloon): handle this.
	}

	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	n.endpoint = endpoint
	n.keystore = ks

	return nil
}

// registerShardingServices sets up either a notary or proposer
// sharding service dependent on the ClientType cli flag
func (n *shardingNode) registerShardingServices() error {
	clientType := n.ctx.GlobalString(utils.ClientType.Name)

	err := n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		if clientType == "notary" {
			return notary.NewNotary(n.ctx)
		}
		return proposer.NewProposer(n.ctx)
	})

	if err != nil {
		return fmt.Errorf("failed to register the main sharding services: %v", err)
	}
	// TODO: registers the shardp2p service.

	return nil
}

// Register appends a struct to the sharding node's services that
// satisfies the Service interface containing lifecycle handlers. Notary, Proposer,
// and ShardP2P are examples of services. The rationale behind this is that the
// sharding node should know very little about the functioning of its underlying
// services as they should be extensible.
func (n *shardingNode) Register(a interface{}) error {
	return nil
}

// Start is the main entrypoint of a sharding node. It starts off every service
// attached to it.
func (n *shardingNode Start() error {
	rpcClient, err := dialRPC(c.endpoint)
	if err != nil {
		return fmt.Errorf("cannot start rpc client. %v", err)
	}
	n.rpcClient = rpcClient
	n.client = ethclient.NewClient(rpcClient)

	// Check account existence and unlock account before starting notary client.
	accounts := n.keystore.Accounts()
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	if err := n.unlockAccount(accounts[0]); err != nil {
		return fmt.Errorf("cannot unlock account. %v", err)
	}

	smc, err := initSMC(c)
	if err != nil {
		return err
	}
	n.smc = smc

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

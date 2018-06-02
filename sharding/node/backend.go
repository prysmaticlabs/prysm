package node

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/rpc"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/notary"
	"github.com/ethereum/go-ethereum/sharding/observer"
	"github.com/ethereum/go-ethereum/sharding/proposer"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/txpool"
	cli "gopkg.in/urfave/cli.v1"
)

var clientIdentifier = "geth"

// ShardEthereum is a service that is registered and started when geth is launched.
// it contains APIs and fields that handle the different components of the sharded
// Ethereum network.
type ShardEthereum struct {
	shardConfig  *params.ShardConfig    // Holds necessary information to configure shards.
	txPool       *txpool.ShardTxPool    // Defines the sharding-specific txpool. To be designed.
	actor        sharding.ShardingActor // Either notary, proposer, or observer.
	shardChainDb ethdb.Database         // Access to the persistent db to store shard data.
	eventFeed    *event.Feed            // Used to enable P2P related interactions via different sharding actors.

	endpoint  string             // Endpoint to JSON RPC.
	client    *ethclient.Client  // Ethereum RPC client.
	keystore  *keystore.KeyStore // Keystore containing the single signer.
	ctx       *cli.Context       // Command line context.
	smc       *contracts.SMC     // The deployed sharding management contract.
	rpcClient *rpc.Client        // The RPC client connection to the main geth node.
	lock      sync.Mutex         // Mutex lock for concurrency management.
}

// New creates a new sharding-enabled Ethereum service. This is called in the main
// geth sharding entrypoint.
func New(ctx *cli.Context) (*ShardEthereum, error) {

	seth := &ShardEthereum{ctx: ctx}

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
		return nil, err
	}

	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	actorFlag := ctx.GlobalString(utils.ActorFlag.Name)

	var actor sharding.ShardingActor

	if actorFlag == "notary" {
		not, err := notary.NewNotary(seth)
		if err != nil {
			return nil, err
		}
		actor = not
	} else if actorFlag == "proposer" {
		prop, err := proposer.NewProposer(seth)
		if err != nil {
			return nil, err
		}
		actor = prop
	} else {
		obs, err := observer.NewObserver(seth)
		if err != nil {
			return nil, err
		}
		actor = obs
	}

	seth.keystore = ks
	seth.endpoint = endpoint
	seth.actor = actor
	return nil, nil
}

// Start the ShardEthereum service and kicks off the p2p and actor's main loop.
func (s *ShardEthereum) Start() error {
	log.Println("Starting sharding service")
	if err := s.actor.Start(); err != nil {
		return err
	}
	defer s.actor.Stop()

	// TODO: start p2p and other relevant services.
	return nil
}

// Close handles graceful shutdown of the system.
func (s *ShardEthereum) Close() error {
	// rpcClient could be nil if the connection failed.
	if s.rpcClient != nil {
		s.rpcClient.Close()
	}
	return nil
}

// CreateTXOpts creates a *TransactOpts with a signer using the default account on the keystore.
func (s *ShardEthereum) CreateTXOpts(value *big.Int) (*bind.TransactOpts, error) {
	account := s.Account()

	return &bind.TransactOpts{
		From:  account.Address,
		Value: value,
		Signer: func(signer types.Signer, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			networkID, err := s.client.NetworkID(context.Background())
			if err != nil {
				return nil, fmt.Errorf("unable to fetch networkID: %v", err)
			}
			return s.keystore.SignTx(*account, tx, networkID /* chainID */)
		},
	}, nil
}

// Account to use for sharding transactions.
func (s *ShardEthereum) Account() *accounts.Account {
	accounts := s.keystore.Accounts()
	return &accounts[0]
}

// Context returns the cli context.
func (s *ShardEthereum) Context() *cli.Context {
	return s.ctx
}

// ChainReader for interacting with the chain.
func (s *ShardEthereum) ChainReader() ethereum.ChainReader {
	return ethereum.ChainReader(s.client)
}

// SMCCaller to interact with the sharding manager contract.
func (s *ShardEthereum) SMCCaller() *contracts.SMCCaller {
	return &s.smc.SMCCaller
}

// SMCTransactor allows us to send tx's to the SMC programmatically.
func (s *ShardEthereum) SMCTransactor() *contracts.SMCTransactor {
	return &s.smc.SMCTransactor
}

// DepositFlagSet returns true for cli flag --deposit.
func (s *ShardEthereum) DepositFlagSet() bool {
	return s.ctx.GlobalBool(utils.DepositFlag.Name)
}

// DataDirFlag returns the datadir flag as a string.
func (s *ShardEthereum) DataDirFlag() string {
	return s.ctx.GlobalString(utils.DataDirFlag.Name)
}

// Client to interact with a geth node via JSON-RPC.
func (s *ShardEthereum) ethereumClient() *ethclient.Client {
	return s.client
}

// unlockAccount will unlock the specified account using utils.PasswordFileFlag
// or empty string if unset.
func (s *ShardEthereum) unlockAccount(account accounts.Account) error {
	pass := ""

	if s.ctx.GlobalIsSet(utils.PasswordFileFlag.Name) {
		file, err := os.Open(s.ctx.GlobalString(utils.PasswordFileFlag.Name))
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

	return s.keystore.Unlock(account, pass)
}

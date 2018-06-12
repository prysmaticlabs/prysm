// Package node defines a backend for a sharding-enabled, Ethereum blockchain.
// It defines a struct which handles the lifecycle of services in the
// sharding system, providing a bridge to the main Ethereum blockchain,
// as well as instantiating peer-to-peer networking for shards.
package node

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/sharding/notary"
	"github.com/ethereum/go-ethereum/sharding/observer"
	shardp2p "github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/proposer"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/txpool"
	cli "gopkg.in/urfave/cli.v1"
)

const shardChainDbName = "shardchaindata"

// ShardEthereum is a service that is registered and started when geth is launched.
// it contains APIs and fields that handle the different components of the sharded
// Ethereum network.
type ShardEthereum struct {
	shardConfig  *params.ShardConfig  // Holds necessary information to configure shards.
	txPool       *txpool.ShardTXPool  // Defines the sharding-specific txpool. To be designed.
	actor        sharding.Actor       // Either notary, proposer, or observer.
	shardChainDb ethdb.Database       // Access to the persistent db to store shard data.
	eventFeed    *event.Feed          // Used to enable P2P related interactions via different sharding actors.
	smcClient    *mainchain.SMCClient // Provides bindings to the SMC deployed on the Ethereum mainchain.

	// Lifecycle and service stores.
	services map[reflect.Type]sharding.Service // Service registry.
	lock     sync.RWMutex
}

// New creates a new sharding-enabled Ethereum instance. This is called in the main
// geth sharding entrypoint.
func New(ctx *cli.Context) (*ShardEthereum, error) {

	shardEthereum := &ShardEthereum{
		services: make(map[reflect.Type]sharding.Service),
	}

	path := node.DefaultDataDir()
	if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		path = ctx.GlobalString(utils.DataDirFlag.Name)
	}

	endpoint := ctx.Args().First()
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s/%s.ipc", path, mainchain.ClientIdentifier)
	}
	if ctx.GlobalIsSet(utils.IPCPathFlag.Name) {
		endpoint = ctx.GlobalString(utils.IPCPathFlag.Name)
	}

	passwordFile := ctx.GlobalString(utils.PasswordFileFlag.Name)
	depositFlag := ctx.GlobalBool(utils.DepositFlag.Name)
	actorFlag := ctx.GlobalString(utils.ActorFlag.Name)
	shardIDFlag := ctx.GlobalInt(utils.ShardIDFlag.Name)

	smcClient, err := mainchain.NewSMCClient(endpoint, path, depositFlag, passwordFile)
	if err != nil {
		return nil, err
	}

	shardChainDb, err := database.NewShardDB(path, shardChainDbName)
	if err != nil {
		return nil, err
	}

	// Adds the initialized SMCClient to the ShardEthereum instance.
	shardEthereum.smcClient = smcClient

	// Adds the initialized shardChainDb to the ShardEthereum instance.
	shardEthereum.shardChainDb = shardChainDb

	if err := shardEthereum.registerP2P(); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerTXPool(actorFlag); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerActorService(actorFlag, shardIDFlag); err != nil {
		return nil, err
	}

	return shardEthereum, nil
}

// Start the ShardEthereum service and kicks off the p2p and actor's main loop.
func (s *ShardEthereum) Start() {

	log.Info("Starting sharding node")

	for _, service := range s.services {
		// Start the next service.
		service.Start()
	}

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		go s.Close()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
		// ensure trace and CPU profile data is flushed.
		debug.Exit()
		debug.LoudPanic("boom")
	}()

	// hang forever...
	select {}
}

// Close handles graceful shutdown of the system.
func (s *ShardEthereum) Close() {
	for kind, service := range s.services {
		if err := service.Stop(); err != nil {
			log.Crit(fmt.Sprintf("Could not stop the following service: %v, %v", kind, err))
		}
	}
	log.Info("Stopping sharding node")
}

// SMCClient returns an instance of a client that communicates to a mainchain node via
// RPC and provides helpful bindings to the Sharding Manager Contract.
func (s *ShardEthereum) SMCClient() *mainchain.SMCClient {
	return s.smcClient
}

// Register appends a service constructor function to the service registry of the
// sharding node.
func (s *ShardEthereum) Register(constructor sharding.ServiceConstructor) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	ctx := &sharding.ServiceContext{
		Services: make(map[reflect.Type]sharding.Service),
	}

	// Copy needed for threaded access.
	for kind, s := range s.services {
		ctx.Services[kind] = s
	}

	service, err := constructor(ctx)
	if err != nil {
		return err
	}

	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		return fmt.Errorf("service already exists: %v", kind)
	}
	s.services[kind] = service

	return nil
}

// registerP2P attaches a shardp2p server to the ShardEthereum instance.
// TODO: Design this shardp2p service and the methods it should expose as well as
// its event loop.
func (s *ShardEthereum) registerP2P() error {
	return s.Register(func(ctx *sharding.ServiceContext) (sharding.Service, error) {
		return shardp2p.NewServer()
	})
}

// registerTXPool is only relevant to proposers in the sharded system. It will
// spin up a transaction pool that will relay incoming transactions via an
// event feed. For our first releases, this can just relay test/fake transaction data
// the proposer can serialize into collation blobs.
// TODO: design this txpool system for our first release.
func (s *ShardEthereum) registerTXPool(actor string) error {
	if actor != "proposer" {
		return nil
	}
	return s.Register(func(ctx *sharding.ServiceContext) (sharding.Service, error) {
		var p2p *shardp2p.Server
		ctx.RetrieveService(&p2p)
		return txpool.NewShardTXPool(p2p)
	})
}

// Registers the actor according to CLI flags. Either notary/proposer/observer.
func (s *ShardEthereum) registerActorService(actor string, shardID int) error {
	return s.Register(func(ctx *sharding.ServiceContext) (sharding.Service, error) {

		var p2p *shardp2p.Server
		ctx.RetrieveService(&p2p)

		if actor == "notary" {
			return notary.NewNotary(s.smcClient, p2p, s.shardChainDb)
		} else if actor == "proposer" {
			var txPool *txpool.ShardTXPool
			ctx.RetrieveService(&txPool)
			return proposer.NewProposer(s.smcClient, p2p, txPool, s.shardChainDb, shardID)
		}
		return observer.NewObserver(p2p, s.shardChainDb, shardID)
	})
}

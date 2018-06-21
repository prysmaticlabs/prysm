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

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/notary"
	"github.com/ethereum/go-ethereum/sharding/observer"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/proposer"
	"github.com/ethereum/go-ethereum/sharding/simulator"
	"github.com/ethereum/go-ethereum/sharding/syncer"
	"github.com/ethereum/go-ethereum/sharding/txpool"
	"gopkg.in/urfave/cli.v1"
)

const shardChainDbName = "shardchaindata"

// ShardEthereum is a service that is registered and started when geth is launched.
// it contains APIs and fields that handle the different components of the sharded
// Ethereum network.
type ShardEthereum struct {
	shardConfig *params.Config // Holds necessary information to configure shards.
	txPool      *txpool.TXPool // Defines the sharding-specific txpool. To be designed.
	actor       sharding.Actor // Either notary, proposer, or observer.
	eventFeed   *event.Feed    // Used to enable P2P related interactions via different sharding actors.

	// Lifecycle and service stores.
	services map[reflect.Type]sharding.Service // Service registry.
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications
}

// New creates a new sharding-enabled Ethereum instance. This is called in the main
// geth sharding entrypoint.
func New(ctx *cli.Context) (*ShardEthereum, error) {
	shardEthereum := &ShardEthereum{
		services: make(map[reflect.Type]sharding.Service),
		stop:     make(chan struct{}),
	}

	// Configure shardConfig by loading the default.
	shardEthereum.shardConfig = params.DefaultConfig

	if err := shardEthereum.registerShardChainDB(ctx); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerP2P(); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerMainchainClient(ctx); err != nil {
		return nil, err
	}

	actorFlag := ctx.GlobalString(utils.ActorFlag.Name)
	if err := shardEthereum.registerTXPool(actorFlag); err != nil {
		return nil, err
	}

	shardIDFlag := ctx.GlobalInt(utils.ShardIDFlag.Name)
	if err := shardEthereum.registerActorService(shardEthereum.shardConfig, actorFlag, shardIDFlag); err != nil {
		return nil, err
	}

	// Should not trigger simulation requests if actor is a notary, as this
	// is supposed to "simulate" notaries sending requests via p2p.
	if actorFlag != "notary" {
		if err := shardEthereum.registerSimulatorService(shardEthereum.shardConfig, shardIDFlag); err != nil {
			return nil, err
		}
	}

	if err := shardEthereum.registerSyncerService(shardEthereum.shardConfig, shardIDFlag); err != nil {
		return nil, err
	}

	return shardEthereum, nil
}

// Start the ShardEthereum service and kicks off the p2p and actor's main loop.
func (s *ShardEthereum) Start() {
	s.lock.Lock()

	log.Info("Starting sharding node")

	for _, service := range s.services {
		// Start the next service.
		service.Start()
	}

	stop := s.stop
	s.lock.Unlock()

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

	// Wait for stop channel to be closed
	<-stop
}

// Close handles graceful shutdown of the system.
func (s *ShardEthereum) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for kind, service := range s.services {
		if err := service.Stop(); err != nil {
			log.Crit(fmt.Sprintf("Could not stop the following service: %v, %v", kind, err))
		}
	}
	log.Info("Stopping sharding node")

	// unblock n.Wait
	close(s.stop)
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

// registerShardChainDB attaches a LevelDB wrapped object to the shardEthereum instance.
func (s *ShardEthereum) registerShardChainDB(ctx *cli.Context) error {
	path := node.DefaultDataDir()
	if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		path = ctx.GlobalString(utils.DataDirFlag.Name)
	}
	return s.Register(func(ctx *sharding.ServiceContext) (sharding.Service, error) {
		return database.NewShardDB(path, shardChainDbName)
	})
}

// registerP2P attaches a p2p server to the ShardEthereum instance.
// TODO: Design this p2p service and the methods it should expose as well as
// its event loop.
func (s *ShardEthereum) registerP2P() error {
	return s.Register(func(ctx *sharding.ServiceContext) (sharding.Service, error) {
		return p2p.NewServer()
	})
}

// registerMainchainClient
func (s *ShardEthereum) registerMainchainClient(ctx *cli.Context) error {
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

	return s.Register(func(ctx *sharding.ServiceContext) (sharding.Service, error) {
		return mainchain.NewSMCClient(endpoint, path, depositFlag, passwordFile)
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
		var p2p *p2p.Server
		ctx.RetrieveService(&p2p)
		return txpool.NewTXPool(p2p)
	})
}

// Registers the actor according to CLI flags. Either notary/proposer/observer.
func (s *ShardEthereum) registerActorService(config *params.Config, actor string, shardID int) error {
	return s.Register(func(ctx *sharding.ServiceContext) (sharding.Service, error) {

		var p2p *p2p.Server
		ctx.RetrieveService(&p2p)
		var smcClient *mainchain.SMCClient
		ctx.RetrieveService(&smcClient)
		var shardChainDB *database.ShardDB
		ctx.RetrieveService(&shardChainDB)

		if actor == "notary" {
			return notary.NewNotary(config, smcClient, p2p, shardChainDB)
		} else if actor == "proposer" {
			var txPool *txpool.TXPool
			ctx.RetrieveService(&txPool)
			return proposer.NewProposer(config, smcClient, p2p, txPool, shardChainDB.DB(), shardID)
		}
		return observer.NewObserver(p2p, shardChainDB.DB(), shardID)
	})
}

func (s *ShardEthereum) registerSimulatorService(config *params.Config, shardID int) error {
	return s.Register(func(ctx *sharding.ServiceContext) (sharding.Service, error) {
		var p2p *p2p.Server
		ctx.RetrieveService(&p2p)
		var smcClient *mainchain.SMCClient
		ctx.RetrieveService(&smcClient)
		return simulator.NewSimulator(config, smcClient, p2p, shardID, 15) // 15 second delay between simulator requests.
	})
}

func (s *ShardEthereum) registerSyncerService(config *params.Config, shardID int) error {
	return s.Register(func(ctx *sharding.ServiceContext) (sharding.Service, error) {
		var p2p *p2p.Server
		ctx.RetrieveService(&p2p)
		var smcClient *mainchain.SMCClient
		ctx.RetrieveService(&smcClient)
		var shardChainDB *database.ShardDB
		ctx.RetrieveService(&shardChainDB)
		return syncer.NewSyncer(config, smcClient, p2p, shardChainDB.DB(), shardID)
	})
}

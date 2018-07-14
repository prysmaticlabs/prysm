// Package node defines a backend for a sharding-enabled, Ethereum blockchain.
// It defines a struct which handles the lifecycle of services in the
// sharding system, providing a bridge to the main Ethereum blockchain,
// as well as instantiating peer-to-peer networking for shards.
package node

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/prysmaticlabs/geth-sharding/sharding/database"
	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	"github.com/prysmaticlabs/geth-sharding/sharding/notary"
	"github.com/prysmaticlabs/geth-sharding/sharding/observer"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
	"github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/proposer"
	"github.com/prysmaticlabs/geth-sharding/sharding/simulator"
	"github.com/prysmaticlabs/geth-sharding/sharding/syncer"
	"github.com/prysmaticlabs/geth-sharding/sharding/txpool"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
	"github.com/prysmaticlabs/geth-sharding/sharding/utils"
	"github.com/prysmaticlabs/geth-sharding/shared"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const shardChainDBName = "shardchaindata"

// ShardEthereum is a service that is registered and started when geth is launched.
// it contains APIs and fields that handle the different components of the sharded
// Ethereum network.
type ShardEthereum struct {
	shardConfig *params.Config // Holds necessary information to configure shards.
	txPool      *txpool.TXPool // Defines the sharding-specific txpool. To be designed.
	actor       types.Actor    // Either notary, proposer, or observer.
	eventFeed   *event.Feed    // Used to enable P2P related interactions via different sharding actors.

	// Lifecycle and service stores.
	services *shared.ServiceRegistry
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications.
}

// New creates a new sharding-enabled Ethereum instance. This is called in the main
// geth sharding entrypoint.
func New(ctx *cli.Context) (*ShardEthereum, error) {
	registry := shared.NewServiceRegistry()
	shardEthereum := &ShardEthereum{
		services: registry,
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

	shardIDFlag := ctx.GlobalInt(utils.ShardIDFlag.Name)
	if err := shardEthereum.registerSyncerService(shardEthereum.shardConfig, shardIDFlag); err != nil {
		return nil, err
	}

	actorFlag := ctx.GlobalString(utils.ActorFlag.Name)
	if err := shardEthereum.registerTXPool(actorFlag); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerActorService(shardEthereum.shardConfig, actorFlag, shardIDFlag); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerSimulatorService(actorFlag, shardEthereum.shardConfig, shardIDFlag); err != nil {
		return nil, err
	}

	return shardEthereum, nil
}

// Start the ShardEthereum service and kicks off the p2p and actor's main loop.
func (s *ShardEthereum) Start() {
	s.lock.Lock()

	log.Info("Starting sharding node")

	s.services.StartAll()

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
				log.Info("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
		panic("Panic closing the sharding node")
	}()

	// Wait for stop channel to be closed.
	<-stop
}

// Close handles graceful shutdown of the system.
func (s *ShardEthereum) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.services.StopAll()
	log.Info("Stopping sharding node")

	close(s.stop)
}

// registerShardChainDB attaches a LevelDB wrapped object to the shardEthereum instance.
func (s *ShardEthereum) registerShardChainDB(ctx *cli.Context) error {
	path := node.DefaultDataDir()
	if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		path = ctx.GlobalString(utils.DataDirFlag.Name)
	}
	shardDB, err := database.NewShardDB(path, shardChainDBName, false)
	if err != nil {
		return fmt.Errorf("could not register shardDB service: %v", err)
	}
	return s.services.RegisterService(shardDB)
}

// registerP2P attaches a p2p server to the ShardEthereum instance.
func (s *ShardEthereum) registerP2P() error {
	shardp2p, err := p2p.NewServer()
	if err != nil {
		return fmt.Errorf("could not register shardp2p service: %v", err)
	}
	return s.services.RegisterService(shardp2p)
}

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

	client, err := mainchain.NewSMCClient(endpoint, path, depositFlag, passwordFile)
	if err != nil {
		return fmt.Errorf("could not register smc client service: %v", err)
	}
	return s.services.RegisterService(client)
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
	var shardp2p *p2p.Server
	if err := s.services.FetchService(&shardp2p); err != nil {
		return err
	}
	pool, err := txpool.NewTXPool(shardp2p)
	if err != nil {
		return fmt.Errorf("could not register shard txpool service: %v", err)
	}
	return s.services.RegisterService(pool)
}

// Registers the actor according to CLI flags. Either notary/proposer/observer.
func (s *ShardEthereum) registerActorService(config *params.Config, actor string, shardID int) error {
	var shardp2p *p2p.Server
	if err := s.services.FetchService(&shardp2p); err != nil {
		return err
	}
	var client *mainchain.SMCClient
	if err := s.services.FetchService(&client); err != nil {
		return err
	}

	var shardChainDB *database.ShardDB
	if err := s.services.FetchService(&shardChainDB); err != nil {
		return err
	}

	var sync *syncer.Syncer
	if err := s.services.FetchService(&sync); err != nil {
		return err
	}

	switch actor {
	case "notary":
		not, err := notary.NewNotary(config, client, shardp2p, shardChainDB)
		if err != nil {
			return fmt.Errorf("could not register notary service: %v", err)
		}
		return s.services.RegisterService(not)
	case "proposer":
		var pool *txpool.TXPool
		if err := s.services.FetchService(&pool); err != nil {
			return err
		}

		prop, err := proposer.NewProposer(config, client, shardp2p, pool, shardChainDB, shardID, sync)
		if err != nil {
			return fmt.Errorf("could not register proposer service: %v", err)
		}
		return s.services.RegisterService(prop)
	case "simulator":
		sim, err := simulator.NewSimulator(config, client, shardp2p, shardID, 15) // 15 second delay between simulator requests.
		if err != nil {
			return fmt.Errorf("could not register simulator service: %v", err)
		}
		return s.services.RegisterService(sim)
	default:
		obs, err := observer.NewObserver(shardp2p, shardChainDB, shardID, sync, client)
		if err != nil {
			return fmt.Errorf("could not register observer service: %v", err)
		}
		return s.services.RegisterService(obs)
	}
}

func (s *ShardEthereum) registerSimulatorService(actorFlag string, config *params.Config, shardID int) error {
	// Should not trigger simulation requests if actor is a notary, as this
	// is supposed to "simulate" notaries sending requests via p2p.
	if actorFlag == "notary" {
		return nil
	}

	var shardp2p *p2p.Server
	if err := s.services.FetchService(&shardp2p); err != nil {
		return err
	}
	var client *mainchain.SMCClient
	if err := s.services.FetchService(&client); err != nil {
		return err
	}

	// 15 second delay between simulator requests.
	sim, err := simulator.NewSimulator(config, client, shardp2p, shardID, 15*time.Second)
	if err != nil {
		return fmt.Errorf("could not register simulator service: %v", err)
	}
	return s.services.RegisterService(sim)
}

func (s *ShardEthereum) registerSyncerService(config *params.Config, shardID int) error {
	var shardp2p *p2p.Server
	if err := s.services.FetchService(&shardp2p); err != nil {
		return err
	}
	var client *mainchain.SMCClient
	if err := s.services.FetchService(&client); err != nil {
		return err
	}

	var shardChainDB *database.ShardDB
	if err := s.services.FetchService(&shardChainDB); err != nil {
		return err
	}

	sync, err := syncer.NewSyncer(config, client, shardp2p, shardChainDB, shardID)
	if err != nil {
		return fmt.Errorf("could not register syncer service: %v", err)
	}
	return s.services.RegisterService(sync)
}

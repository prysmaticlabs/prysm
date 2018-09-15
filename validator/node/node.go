// Package node defines a validator node which connects to a
// full beacon node as part of the Ethereum 2.0 specification.
package node

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/validator/attester"
	"github.com/prysmaticlabs/prysm/validator/beacon"
	"github.com/prysmaticlabs/prysm/validator/proposer"
	"github.com/prysmaticlabs/prysm/validator/rpcclient"
	"github.com/prysmaticlabs/prysm/validator/txpool"
	"github.com/prysmaticlabs/prysm/validator/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "node")

const shardChainDBName = "shardchaindata"

// ShardEthereum defines an instance of a sharding validator that manages
// the entire lifecycle of services attached to it participating in
// Ethereum 2.0.
type ShardEthereum struct {
	services *shared.ServiceRegistry // Lifecycle and service store.
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications.
	db       *database.DB
}

// NewShardInstance creates a new, Ethereum 2.0 sharding validator.
func NewShardInstance(ctx *cli.Context) (*ShardEthereum, error) {
	registry := shared.NewServiceRegistry()
	shardEthereum := &ShardEthereum{
		services: registry,
		stop:     make(chan struct{}),
	}

	if err := shardEthereum.startDB(ctx); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerP2P(); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerTXPool(); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerRPCClientService(ctx); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerBeaconService(); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerAttesterService(); err != nil {
		return nil, err
	}

	if err := shardEthereum.registerProposerService(); err != nil {
		return nil, err
	}

	return shardEthereum, nil
}

// Start every service in the sharding validator.
func (s *ShardEthereum) Start() {
	s.lock.Lock()

	log.Info("Starting sharding validator")

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
		debug.Exit() // Ensure trace and CPU profile data are flushed.
		panic("Panic closing the sharding validator")
	}()

	// Wait for stop channel to be closed.
	<-stop
}

// Close handles graceful shutdown of the system.
func (s *ShardEthereum) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.db.Close()
	s.services.StopAll()
	log.Info("Stopping sharding validator")

	close(s.stop)
}

// startDB attaches a LevelDB wrapped object to the shardEthereum instance.
func (s *ShardEthereum) startDB(ctx *cli.Context) error {
	path := ctx.GlobalString(cmd.DataDirFlag.Name)
	config := &database.DBConfig{DataDir: path, Name: shardChainDBName, InMemory: false}
	db, err := database.NewDB(config)
	if err != nil {
		return err
	}

	s.db = db
	return nil
}

// registerP2P attaches a p2p server to the ShardEthereum instance.
func (s *ShardEthereum) registerP2P() error {
	shardp2p, err := configureP2P()
	if err != nil {
		return fmt.Errorf("could not register shardp2p service: %v", err)
	}
	return s.services.RegisterService(shardp2p)
}

// registerTXPool creates a service that
// can spin up a transaction pool that will relay incoming transactions via an
// event feed. For our first releases, this can just relay test/fake transaction data
// the proposer can serialize into collation blobs.
// TODO: design this txpool system for our first release.
func (s *ShardEthereum) registerTXPool() error {
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

// registerBeaconService registers a service that fetches streams from a beacon node
// via RPC.
func (s *ShardEthereum) registerBeaconService() error {
	var rpcService *rpcclient.Service
	if err := s.services.FetchService(&rpcService); err != nil {
		return err
	}
	b := beacon.NewBeaconValidator(context.TODO(), rpcService)
	return s.services.RegisterService(b)
}

// registerAttesterService that listens to assignments from the beacon service.
func (s *ShardEthereum) registerAttesterService() error {
	var beaconService *beacon.Service
	if err := s.services.FetchService(&beaconService); err != nil {
		return err
	}

	att := attester.NewAttester(context.TODO(), &attester.Config{
		Assigner:      beaconService,
		AssignmentBuf: 100,
	})
	return s.services.RegisterService(att)
}

// registerProposerService that listens to assignments from the beacon service.
func (s *ShardEthereum) registerProposerService() error {
	var rpcService *rpcclient.Service
	if err := s.services.FetchService(&rpcService); err != nil {
		return err
	}
	var beaconService *beacon.Service
	if err := s.services.FetchService(&beaconService); err != nil {
		return err
	}

	prop := proposer.NewProposer(context.TODO(), &proposer.Config{
		Assigner:              beaconService,
		Client:                rpcService,
		AssignmentBuf:         100,
		AttestationBufferSize: 100,
		AttesterFeed:          beaconService,
	})
	return s.services.RegisterService(prop)
}

// registerRPCClientService registers a new RPC client that connects to a beacon node.
func (s *ShardEthereum) registerRPCClientService(ctx *cli.Context) error {
	endpoint := ctx.GlobalString(types.BeaconRPCProviderFlag.Name)
	rpcService := rpcclient.NewRPCClient(context.TODO(), &rpcclient.Config{
		Endpoint: endpoint,
	})
	return s.services.RegisterService(rpcService)
}

// Package node defines a validator client which connects to a
// full beacon node as part of the Ethereum Serenity specification.
package node

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/version"
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

// ValidatorClient defines an instance of a sharding validator that manages
// the entire lifecycle of services attached to it participating in
// Ethereum Serenity.
type ValidatorClient struct {
	ctx      *cli.Context
	services *shared.ServiceRegistry // Lifecycle and service store.
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications.
	db       *database.DB
}

// GeneratePubKey generates a random public key for the validator, if they have not provided one.
func GeneratePubKey() ([]byte, error) {
	pubkey := make([]byte, 48)
	_, err := rand.Read(pubkey)

	return pubkey, err
}

// NewValidatorClient creates a new, Ethereum Serenity validator client.
func NewValidatorClient(ctx *cli.Context) (*ValidatorClient, error) {
	registry := shared.NewServiceRegistry()
	ValidatorClient := &ValidatorClient{
		ctx:      ctx,
		services: registry,
		stop:     make(chan struct{}),
	}

	var pubKey []byte

	keystorePath := ctx.GlobalString(cmd.KeystoreDirectoryFlag.Name)
	password := ctx.GlobalString(cmd.KeystorePasswordFlag.Name)
	if keystorePath != "" && password != "" {
		blspubkey, err := keystore.RetrievePubKey(keystorePath, password)
		if err != nil {
			return nil, err
		}

		pubKey = blspubkey.BufferedPublicKey()
	} else {
		pubKey = []byte(ctx.GlobalString(types.PubKeyFlag.Name))
	}

	if err := ValidatorClient.startDB(ctx); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerP2P(ctx); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerTXPool(); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerRPCClientService(ctx); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerBeaconService(pubKey); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerAttesterService(pubKey); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerProposerService(); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerPrometheusService(ctx); err != nil {
		return nil, err
	}

	return ValidatorClient, nil
}

// Start every service in the validator client.
func (s *ValidatorClient) Start() {
	s.lock.Lock()

	log.WithFields(logrus.Fields{
		"version": version.GetVersion(),
	}).Info("Starting validator node")

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
		debug.Exit(s.ctx) // Ensure trace and CPU profile data are flushed.
		panic("Panic closing the sharding validator")
	}()

	// Wait for stop channel to be closed.
	<-stop
}

// Close handles graceful shutdown of the system.
func (s *ValidatorClient) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.db.Close()
	s.services.StopAll()
	log.Info("Stopping sharding validator")

	close(s.stop)
}

// startDB attaches a LevelDB wrapped object to the ValidatorClient instance.
func (s *ValidatorClient) startDB(ctx *cli.Context) error {
	path := ctx.GlobalString(cmd.DataDirFlag.Name)
	config := &database.DBConfig{DataDir: path, Name: shardChainDBName, InMemory: false}
	db, err := database.NewDB(config)
	if err != nil {
		return err
	}

	s.db = db
	return nil
}

// registerP2P attaches a p2p server to the ValidatorClient instance.
func (s *ValidatorClient) registerP2P(ctx *cli.Context) error {
	shardp2p, err := configureP2P(ctx)
	if err != nil {
		return fmt.Errorf("could not register shardp2p service: %v", err)
	}
	return s.services.RegisterService(shardp2p)
}

// registerTXPool creates a service that
// can spin up a transaction pool that will relay incoming transactions via an
// event feed. For our first releases, this can just relay test/fake transaction data
// the proposer can serialize into collation blobs.
// TODO(#161): design this txpool system for our first release.
func (s *ValidatorClient) registerTXPool() error {
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
func (s *ValidatorClient) registerBeaconService(pubKey []byte) error {
	var rpcService *rpcclient.Service
	if err := s.services.FetchService(&rpcService); err != nil {
		return err
	}
	b := beacon.NewBeaconValidator(context.TODO(), pubKey, rpcService)
	return s.services.RegisterService(b)
}

// registerAttesterService that listens to assignments from the beacon service.
func (s *ValidatorClient) registerAttesterService(pubKey []byte) error {
	var beaconService *beacon.Service
	if err := s.services.FetchService(&beaconService); err != nil {
		return err
	}

	var rpcService *rpcclient.Service
	if err := s.services.FetchService(&rpcService); err != nil {
		return err
	}

	att := attester.NewAttester(context.TODO(), &attester.Config{
		Assigner:      beaconService,
		AssignmentBuf: 100,
		Client:        rpcService,
		PublicKey:     pubKey,
	})
	return s.services.RegisterService(att)
}

// registerProposerService that listens to assignments from the beacon service.
func (s *ValidatorClient) registerProposerService() error {
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
func (s *ValidatorClient) registerRPCClientService(ctx *cli.Context) error {
	endpoint := ctx.GlobalString(types.BeaconRPCProviderFlag.Name)
	rpcService := rpcclient.NewRPCClient(context.TODO(), &rpcclient.Config{
		Endpoint: endpoint,
	})
	return s.services.RegisterService(rpcService)
}

func (s *ValidatorClient) registerPrometheusService(ctx *cli.Context) error {
	service := prometheus.NewPrometheusService(
		fmt.Sprintf(":%d", ctx.GlobalInt64(cmd.MonitoringPortFlag.Name)),
		s.services,
	)
	logrus.AddHook(prometheus.NewLogrusCollector())
	return s.services.RegisterService(service)
}

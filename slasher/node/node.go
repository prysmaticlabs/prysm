package node

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "node")

const beaconChainDBName = "beaconchaindata"
const testSkipPowFlag = "test-skip-pow"

// SlasherNode defines a struct that handles the services running a slashing detector
// for eth2. It handles the lifecycle of the entire system and registers
// services to a service registry.
type SlasherNode struct {
	ctx             *cli.Context
	lock            sync.RWMutex
	services        *shared.ServiceRegistry
	stop            chan struct{} // Channel to wait for termination notifications.
	db              db.Database
	attestationFeed *event.Feed
	blockFeed       *event.Feed
}

// NewSlasherNode creates a new node instance, sets up configuration options,
// and registers every required service.
func NewSlasherNode(ctx *cli.Context) (*SlasherNode, error) {
	if err := tracing.Setup(
		"slasher", // Service name.
		ctx.GlobalString(cmd.TracingProcessNameFlag.Name),
		ctx.GlobalString(cmd.TracingEndpointFlag.Name),
		ctx.GlobalFloat64(cmd.TraceSampleFractionFlag.Name),
		ctx.GlobalBool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}
	registry := shared.NewServiceRegistry()

	// Use custom config values if the --no-custom-config flag is not set.
	if !ctx.GlobalBool(flags.NoCustomConfigFlag.Name) {
		if featureconfig.Get().MinimalConfig {
			log.WithField(
				"config", "minimal-spec",
			).Info("Using custom chain parameters")
			params.UseMinimalConfig()
		} else {
			log.WithField(
				"config", "demo",
			).Info("Using custom chain parameters")
			params.UseDemoBeaconConfig()
		}
	}

	slasher := &SlasherNode{
		ctx:             ctx,
		services:        registry,
		stop:            make(chan struct{}),
		attestationFeed: new(event.Feed),
		blockFeed:       new(event.Feed),
	}

	//if err := slasher.startDB(ctx); err != nil {
	//	return nil, err
	//}
	return slasher, nil
}

// Start the slasher and kick off every registered service.
func (s *SlasherNode) Start() {
	s.lock.Lock()
	s.services.StartAll()
	s.lock.Unlock()

	stop := s.stop
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		debug.Exit(s.ctx) // Ensure trace and CPU profile data are flushed.
		go s.Close()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Info("Already shutting down, interrupt more to panic", "times", i-1)
			}
		}
		panic("Panic closing the beacon node")
	}()

	// Wait for stop channel to be closed.
	<-stop
}

// Close handles graceful shutdown of the system.
func (s *SlasherNode) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Info("Stopping hash slinging slasher")
	s.services.StopAll()
	if err := s.db.Close(); err != nil {
		log.Errorf("Failed to close database: %v", err)
	}
	close(s.stop)
}

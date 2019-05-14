// Package node defines a validator client which connects to a
// full beacon node as part of the Ethereum Serenity specification.
package node

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/validator/client"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "node")

const validatorDBName = "validatordata"

// ValidatorClient defines an instance of a sharding validator that manages
// the entire lifecycle of services attached to it participating in
// Ethereum Serenity.
type ValidatorClient struct {
	ctx      *cli.Context
	services *shared.ServiceRegistry // Lifecycle and service store.
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications.
	db       *db.ValidatorDB
}

// NewValidatorClient creates a new, Ethereum Serenity validator client.
func NewValidatorClient(ctx *cli.Context, password string) (*ValidatorClient, error) {
	if err := tracing.Setup(
		"validator", // service name
		ctx.GlobalString(cmd.TracingProcessNameFlag.Name),
		ctx.GlobalString(cmd.TracingEndpointFlag.Name),
		ctx.GlobalFloat64(cmd.TraceSampleFractionFlag.Name),
		ctx.GlobalBool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}
	registry := shared.NewServiceRegistry()
	ValidatorClient := &ValidatorClient{
		ctx:      ctx,
		services: registry,
		stop:     make(chan struct{}),
	}

	// Use custom config values if the --no-custom-config flag is set.
	if !ctx.GlobalBool(types.NoCustomConfigFlag.Name) {
		log.Info("Using custom parameter configuration")
		params.UseDemoBeaconConfig()
	}

	featureconfig.ConfigureBeaconFeatures(ctx)

	if err := ValidatorClient.startDB(ctx); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerPrometheusService(ctx); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerClientService(ctx, password); err != nil {
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
		debug.Exit(s.ctx) // Ensure trace and CPU profile data are flushed.
		go s.Close()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Info("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
		panic("Panic closing the sharding validator")
	}()

	// Wait for stop channel to be closed.
	<-stop
}

// Close handles graceful shutdown of the system.
func (s *ValidatorClient) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.services.StopAll()
	log.Info("Stopping sharding validator")

	if err := s.db.Close(); err != nil {
		log.Errorf("Failed to close database: %v", err)
	}

	close(s.stop)
}

func (s *ValidatorClient) startDB(ctx *cli.Context) error {
	baseDir := ctx.GlobalString(cmd.DataDirFlag.Name)

	db, err := db.NewDB(path.Join(baseDir, validatorDBName))
	if err != nil {
		return err
	}

	log.Info("checking db")
	s.db = db
	return nil
}

func (s *ValidatorClient) registerPrometheusService(ctx *cli.Context) error {
	service := prometheus.NewPrometheusService(
		fmt.Sprintf(":%d", ctx.GlobalInt64(cmd.MonitoringPortFlag.Name)),
		s.services,
	)
	logrus.AddHook(prometheus.NewLogrusCollector())
	return s.services.RegisterService(service)
}

func (s *ValidatorClient) registerClientService(ctx *cli.Context, password string) error {
	endpoint := ctx.GlobalString(types.BeaconRPCProviderFlag.Name)
	keystoreDirectory := ctx.GlobalString(types.KeystorePathFlag.Name)
	logValidatorBalances := !ctx.GlobalBool(types.DisablePenaltyRewardLogFlag.Name)
	v, err := client.NewValidatorService(context.Background(), &client.Config{
		Endpoint:             endpoint,
		KeystorePath:         keystoreDirectory,
		Password:             password,
		LogValidatorBalances: logValidatorBalances,
		ValidatorDB:          s.db,
	})
	if err != nil {
		return fmt.Errorf("could not initialize client service: %v", err)
	}
	return s.services.RegisterService(v)
}

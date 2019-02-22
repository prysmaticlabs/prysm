// Package node defines a validator client which connects to a
// full beacon node as part of the Ethereum Serenity specification.
package node

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prysmaticlabs/prysm/validator/types"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/validator/client"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "node")

// ValidatorClient defines an instance of a sharding validator that manages
// the entire lifecycle of services attached to it participating in
// Ethereum Serenity.
type ValidatorClient struct {
	ctx      *cli.Context
	services *shared.ServiceRegistry // Lifecycle and service store.
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications.
}

// NewValidatorClient creates a new, Ethereum Serenity validator client.
func NewValidatorClient(ctx *cli.Context) (*ValidatorClient, error) {
	registry := shared.NewServiceRegistry()
	ValidatorClient := &ValidatorClient{
		ctx:      ctx,
		services: registry,
		stop:     make(chan struct{}),
	}

	if err := ValidatorClient.registerPrometheusService(ctx); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerClientService(ctx); err != nil {
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

	s.services.StopAll()
	log.Info("Stopping sharding validator")

	close(s.stop)
}

func (s *ValidatorClient) registerPrometheusService(ctx *cli.Context) error {
	service := prometheus.NewPrometheusService(
		fmt.Sprintf(":%d", ctx.GlobalInt64(cmd.MonitoringPortFlag.Name)),
		s.services,
	)
	logrus.AddHook(prometheus.NewLogrusCollector())
	return s.services.RegisterService(service)
}

func (s *ValidatorClient) registerClientService(ctx *cli.Context) error {
	endpoint := ctx.GlobalString(types.BeaconRPCProviderFlag.Name)
	keystoreDirectory := ctx.GlobalString(types.KeystorePathFlag.Name)
	keystorePassword := ctx.GlobalString(types.PasswordFlag.Name)
	v, err := client.NewValidatorService(context.TODO(), &client.Config{
		Endpoint:     endpoint,
		KeystorePath: keystoreDirectory,
		Password:     keystorePassword,
	})
	if err != nil {
		return fmt.Errorf("could not initialize client service: %v", err)
	}
	return s.services.RegisterService(v)
}

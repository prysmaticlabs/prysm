// Package node is the main process which handles the lifecycle of
// the runtime services in a slasher process, gracefully shutting
// everything down upon close.
package node

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
	"github.com/prysmaticlabs/prysm/slasher/detection"
	"github.com/prysmaticlabs/prysm/slasher/flags"
	"github.com/prysmaticlabs/prysm/slasher/rpc"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "node")

const slasherDBName = "slasherdata"

// SlasherNode defines a struct that handles the services running a slashing detector
// for eth2. It handles the lifecycle of the entire system and registers
// services to a service registry.
type SlasherNode struct {
	cliCtx                *cli.Context
	ctx                   context.Context
	cancel                context.CancelFunc
	lock                  sync.RWMutex
	services              *shared.ServiceRegistry
	proposerSlashingsFeed *event.Feed
	attesterSlashingsFeed *event.Feed
	stop                  chan struct{} // Channel to wait for termination notifications.
	db                    db.Database
}

// NewSlasherNode creates a new node instance, sets up configuration options,
// and registers every required service.
func NewSlasherNode(cliCtx *cli.Context) (*SlasherNode, error) {
	if err := tracing.Setup(
		"slasher", // Service name.
		cliCtx.String(cmd.TracingProcessNameFlag.Name),
		cliCtx.String(cmd.TracingEndpointFlag.Name),
		cliCtx.Float64(cmd.TraceSampleFractionFlag.Name),
		cliCtx.Bool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}

	cmd.ConfigureSlasher(cliCtx)
	featureconfig.ConfigureSlasher(cliCtx)
	registry := shared.NewServiceRegistry()

	ctx, cancel := context.WithCancel(context.Background())
	slasher := &SlasherNode{
		cliCtx:                cliCtx,
		ctx:                   ctx,
		cancel:                cancel,
		proposerSlashingsFeed: new(event.Feed),
		attesterSlashingsFeed: new(event.Feed),
		services:              registry,
		stop:                  make(chan struct{}),
	}
	if err := slasher.registerPrometheusService(); err != nil {
		return nil, err
	}

	if err := slasher.startDB(); err != nil {
		return nil, err
	}

	if err := slasher.registerBeaconClientService(); err != nil {
		return nil, err
	}

	if err := slasher.registerDetectionService(); err != nil {
		return nil, err
	}

	if err := slasher.registerRPCService(); err != nil {
		return nil, err
	}

	return slasher, nil
}

// Start the slasher and kick off every registered service.
func (s *SlasherNode) Start() {
	s.lock.Lock()
	s.services.StartAll()
	s.lock.Unlock()

	log.WithFields(logrus.Fields{
		"version": version.GetVersion(),
	}).Info("Starting slasher client")

	stop := s.stop
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		debug.Exit(s.cliCtx) // Ensure trace and CPU profile data are flushed.
		go s.Close()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.WithField("times", i-1).Info("Already shutting down, interrupt more to panic")
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
	s.cancel()
	s.services.StopAll()
	if err := s.db.Close(); err != nil {
		log.Errorf("Failed to close database: %v", err)
	}
	close(s.stop)
}

func (s *SlasherNode) registerPrometheusService() error {
	service := prometheus.NewPrometheusService(
		fmt.Sprintf("%s:%d", s.cliCtx.String(cmd.MonitoringHostFlag.Name), s.cliCtx.Int(flags.MonitoringPortFlag.Name)),
		s.services,
	)
	logrus.AddHook(prometheus.NewLogrusCollector())
	return s.services.RegisterService(service)
}

func (s *SlasherNode) startDB() error {
	baseDir := s.cliCtx.String(cmd.DataDirFlag.Name)
	clearDB := s.cliCtx.Bool(cmd.ClearDB.Name)
	forceClearDB := s.cliCtx.Bool(cmd.ForceClearDB.Name)
	dbPath := path.Join(baseDir, slasherDBName)
	cfg := &kv.Config{}
	d, err := db.NewDB(dbPath, cfg)
	if err != nil {
		return err
	}
	clearDBConfirmed := false
	if clearDB && !forceClearDB {
		actionText := "This will delete your slasher database stored in your data directory. " +
			"Your database backups will not be removed - do you want to proceed? (Y/N)"
		deniedText := "Database will not be deleted. No changes have been made."
		clearDBConfirmed, err = cmd.ConfirmAction(actionText, deniedText)
		if err != nil {
			return err
		}
	}
	if clearDBConfirmed || forceClearDB {
		log.Warning("Removing database")
		if err := d.ClearDB(); err != nil {
			return err
		}
		d, err = db.NewDB(dbPath, cfg)
		if err != nil {
			return err
		}
	}
	log.WithField("database-path", baseDir).Info("Checking DB")
	s.db = d
	return nil
}

func (s *SlasherNode) registerBeaconClientService() error {
	beaconCert := s.cliCtx.String(flags.BeaconCertFlag.Name)
	beaconProvider := s.cliCtx.String(flags.BeaconRPCProviderFlag.Name)
	if beaconProvider == "" {
		beaconProvider = flags.BeaconRPCProviderFlag.Value
	}

	bs, err := beaconclient.NewBeaconClientService(s.ctx, &beaconclient.Config{
		BeaconCert:            beaconCert,
		SlasherDB:             s.db,
		BeaconProvider:        beaconProvider,
		AttesterSlashingsFeed: s.attesterSlashingsFeed,
		ProposerSlashingsFeed: s.proposerSlashingsFeed,
	})
	if err != nil {
		return errors.Wrap(err, "failed to initialize beacon client")
	}
	return s.services.RegisterService(bs)
}

func (s *SlasherNode) registerDetectionService() error {
	var bs *beaconclient.Service
	if err := s.services.FetchService(&bs); err != nil {
		panic(err)
	}
	ds := detection.NewDetectionService(s.ctx, &detection.Config{
		Notifier:              bs,
		SlasherDB:             s.db,
		BeaconClient:          bs,
		ChainFetcher:          bs,
		AttesterSlashingsFeed: s.attesterSlashingsFeed,
		ProposerSlashingsFeed: s.proposerSlashingsFeed,
	})
	return s.services.RegisterService(ds)
}

func (s *SlasherNode) registerRPCService() error {
	var detectionService *detection.Service
	if err := s.services.FetchService(&detectionService); err != nil {
		return err
	}
	var bs *beaconclient.Service
	if err := s.services.FetchService(&bs); err != nil {
		panic(err)
	}
	host := s.cliCtx.String(flags.RPCHost.Name)
	port := s.cliCtx.String(flags.RPCPort.Name)
	cert := s.cliCtx.String(flags.CertFlag.Name)
	key := s.cliCtx.String(flags.KeyFlag.Name)
	rpcService := rpc.NewService(s.ctx, &rpc.Config{
		Host:         host,
		Port:         port,
		CertFlag:     cert,
		KeyFlag:      key,
		Detector:     detectionService,
		SlasherDB:    s.db,
		BeaconClient: bs,
	})

	return s.services.RegisterService(rpcService)
}

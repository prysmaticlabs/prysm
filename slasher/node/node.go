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
	"github.com/prysmaticlabs/prysm/cmd/slasher/flags"
	"github.com/prysmaticlabs/prysm/runtime/debug"
	"github.com/prysmaticlabs/prysm/runtime/prereqs"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/backuputil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
	"github.com/prysmaticlabs/prysm/slasher/detection"
	"github.com/prysmaticlabs/prysm/slasher/rpc"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// SlasherNode defines a struct that handles the services running a slashing detector
// for Ethereum. It handles the lifecycle of the entire system and registers
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

// New creates a new node instance, sets up configuration options,
// and registers every required service.
func New(cliCtx *cli.Context) (*SlasherNode, error) {
	if err := tracing.Setup(
		"slasher", // Service name.
		cliCtx.String(cmd.TracingProcessNameFlag.Name),
		cliCtx.String(cmd.TracingEndpointFlag.Name),
		cliCtx.Float64(cmd.TraceSampleFractionFlag.Name),
		cliCtx.Bool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}

	// Warn if user's platform is not supported
	prereqs.WarnIfPlatformNotSupported(cliCtx.Context)

	if cliCtx.Bool(flags.EnableHistoricalDetectionFlag.Name) {
		// Set the max RPC size to 4096 as configured by --historical-slasher-node for optimal historical detection.
		cmdConfig := cmd.Get()
		cmdConfig.MaxRPCPageSize = int(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations))
		cmd.Init(cmdConfig)
	}

	featureconfig.ConfigureSlasher(cliCtx)
	cmd.ConfigureSlasher(cliCtx)
	registry := shared.NewServiceRegistry()

	ctx, cancel := context.WithCancel(cliCtx.Context)
	slasher := &SlasherNode{
		cliCtx:                cliCtx,
		ctx:                   ctx,
		cancel:                cancel,
		proposerSlashingsFeed: new(event.Feed),
		attesterSlashingsFeed: new(event.Feed),
		services:              registry,
		stop:                  make(chan struct{}),
	}

	if err := slasher.startDB(); err != nil {
		return nil, err
	}

	if !cliCtx.Bool(cmd.DisableMonitoringFlag.Name) {
		if err := slasher.registerPrometheusService(cliCtx); err != nil {
			return nil, err
		}
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
func (n *SlasherNode) Start() {
	n.lock.Lock()
	n.services.StartAll()
	n.lock.Unlock()

	log.WithFields(logrus.Fields{
		"version": version.Version(),
	}).Info("Starting slasher client")

	stop := n.stop
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		debug.Exit(n.cliCtx) // Ensure trace and CPU profile data are flushed.
		go n.Close()
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
func (n *SlasherNode) Close() {
	n.lock.Lock()
	defer n.lock.Unlock()

	log.Info("Stopping hash slinging slasher")
	n.services.StopAll()
	if err := n.db.Close(); err != nil {
		log.Errorf("Failed to close database: %v", err)
	}
	n.cancel()
	close(n.stop)
}

func (n *SlasherNode) registerPrometheusService(cliCtx *cli.Context) error {
	var additionalHandlers []prometheus.Handler
	if cliCtx.IsSet(cmd.EnableBackupWebhookFlag.Name) {
		additionalHandlers = append(
			additionalHandlers,
			prometheus.Handler{
				Path:    "/db/backup",
				Handler: backuputil.BackupHandler(n.db, cliCtx.String(cmd.BackupWebhookOutputDir.Name)),
			},
		)
	}
	service := prometheus.NewService(
		fmt.Sprintf("%s:%d", n.cliCtx.String(cmd.MonitoringHostFlag.Name), n.cliCtx.Int(flags.MonitoringPortFlag.Name)),
		n.services,
		additionalHandlers...,
	)
	logrus.AddHook(prometheus.NewLogrusCollector())
	return n.services.RegisterService(service)
}

func (n *SlasherNode) startDB() error {
	baseDir := n.cliCtx.String(cmd.DataDirFlag.Name)
	clearDB := n.cliCtx.Bool(cmd.ClearDB.Name)
	forceClearDB := n.cliCtx.Bool(cmd.ForceClearDB.Name)
	dbPath := path.Join(baseDir, kv.SlasherDbDirName)
	spanCacheSize := n.cliCtx.Int(flags.SpanCacheSize.Name)
	highestAttCacheSize := n.cliCtx.Int(flags.HighestAttCacheSize.Name)
	cfg := &kv.Config{SpanCacheSize: spanCacheSize, HighestAttestationCacheSize: highestAttCacheSize}
	log.Infof("Span cache size has been set to: %d", spanCacheSize)
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
		if err := d.Close(); err != nil {
			return errors.Wrap(err, "could not close db prior to clearing")
		}
		if err := d.ClearDB(); err != nil {
			return err
		}
		d, err = db.NewDB(dbPath, cfg)
		if err != nil {
			return err
		}
	}
	log.WithField("database-path", baseDir).Info("Checking DB")
	n.db = d
	return nil
}

func (n *SlasherNode) registerBeaconClientService() error {
	beaconCert := n.cliCtx.String(flags.BeaconCertFlag.Name)
	beaconProvider := n.cliCtx.String(flags.BeaconRPCProviderFlag.Name)
	if beaconProvider == "" {
		beaconProvider = flags.BeaconRPCProviderFlag.Value
	}

	bs, err := beaconclient.NewService(n.ctx, &beaconclient.Config{
		BeaconCert:            beaconCert,
		SlasherDB:             n.db,
		BeaconProvider:        beaconProvider,
		AttesterSlashingsFeed: n.attesterSlashingsFeed,
		ProposerSlashingsFeed: n.proposerSlashingsFeed,
	})
	if err != nil {
		return errors.Wrap(err, "failed to initialize beacon client")
	}
	return n.services.RegisterService(bs)
}

func (n *SlasherNode) registerDetectionService() error {
	var bs *beaconclient.Service
	if err := n.services.FetchService(&bs); err != nil {
		panic(err)
	}
	ds := detection.NewService(n.ctx, &detection.Config{
		Notifier:              bs,
		SlasherDB:             n.db,
		BeaconClient:          bs,
		ChainFetcher:          bs,
		AttesterSlashingsFeed: n.attesterSlashingsFeed,
		ProposerSlashingsFeed: n.proposerSlashingsFeed,
		HistoricalDetection:   n.cliCtx.Bool(flags.EnableHistoricalDetectionFlag.Name),
	})
	return n.services.RegisterService(ds)
}

func (n *SlasherNode) registerRPCService() error {
	var detectionService *detection.Service
	if err := n.services.FetchService(&detectionService); err != nil {
		return err
	}
	var bs *beaconclient.Service
	if err := n.services.FetchService(&bs); err != nil {
		panic(err)
	}
	host := n.cliCtx.String(flags.RPCHost.Name)
	port := n.cliCtx.String(flags.RPCPort.Name)
	cert := n.cliCtx.String(flags.CertFlag.Name)
	key := n.cliCtx.String(flags.KeyFlag.Name)
	rpcService := rpc.NewService(n.ctx, &rpc.Config{
		Host:         host,
		Port:         port,
		CertFlag:     cert,
		KeyFlag:      key,
		Detector:     detectionService,
		SlasherDB:    n.db,
		BeaconClient: bs,
	})

	return n.services.RegisterService(rpcService)
}

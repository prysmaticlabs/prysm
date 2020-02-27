package node

import (
	"context"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
	"github.com/prysmaticlabs/prysm/slasher/detection"
	"github.com/prysmaticlabs/prysm/slasher/flags"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "node")

const slasherDBName = "slasherdata"

// SlasherNode defines a struct that handles the services running a slashing detector
// for eth2. It handles the lifecycle of the entire system and registers
// services to a service registry.
type SlasherNode struct {
	ctx                   *cli.Context
	lock                  sync.RWMutex
	services              *shared.ServiceRegistry
	proposerSlashingsFeed *event.Feed
	attesterSlashingsFeed *event.Feed
	stop                  chan struct{} // Channel to wait for termination notifications.
	db                    db.Database
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

	slasher := &SlasherNode{
		ctx:                   ctx,
		proposerSlashingsFeed: new(event.Feed),
		attesterSlashingsFeed: new(event.Feed),
		services:              registry,
		stop:                  make(chan struct{}),
	}

	if err := slasher.startDB(ctx); err != nil {
		return nil, err
	}

	if err := slasher.registerBeaconClientService(ctx); err != nil {
		return nil, err
	}

	if err := slasher.registerDetectionService(); err != nil {
		return nil, err
	}

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

func (s *SlasherNode) startDB(ctx *cli.Context) error {
	baseDir := ctx.GlobalString(cmd.DataDirFlag.Name)
	clearDB := ctx.GlobalBool(cmd.ClearDB.Name)
	forceClearDB := ctx.GlobalBool(cmd.ForceClearDB.Name)
	dbPath := path.Join(baseDir, slasherDBName)
	cfg := &kv.Config{SpanCacheEnabled: ctx.GlobalBool(flags.UseSpanCacheFlag.Name)}
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

func (s *SlasherNode) registerBeaconClientService(ctx *cli.Context) error {
	beaconCert := ctx.GlobalString(flags.BeaconCertFlag.Name)
	beaconProvider := ctx.GlobalString(flags.BeaconRPCProviderFlag.Name)
	if beaconProvider == "" {
		beaconProvider = flags.BeaconRPCProviderFlag.Value
	}

	bs := beaconclient.NewBeaconClientService(context.Background(), &beaconclient.Config{
		BeaconCert:            beaconCert,
		SlasherDB:             s.db,
		BeaconProvider:        beaconProvider,
		AttesterSlashingsFeed: s.attesterSlashingsFeed,
		ProposerSlashingsFeed: s.proposerSlashingsFeed,
	})
	return s.services.RegisterService(bs)
}

func (s *SlasherNode) registerDetectionService() error {
	var bs *beaconclient.Service
	if err := s.services.FetchService(&bs); err != nil {
		panic(err)
	}
	ds := detection.NewDetectionService(context.Background(), &detection.Config{
		Notifier:              bs,
		SlasherDB:             s.db,
		BeaconClient:          bs,
		ChainFetcher:          bs,
		AttesterSlashingsFeed: s.attesterSlashingsFeed,
		ProposerSlashingsFeed: s.proposerSlashingsFeed,
	})
	return s.services.RegisterService(ds)
}

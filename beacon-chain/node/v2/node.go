package v2

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/runtime"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "node")

type Option func(*BeaconNode) error

type BeaconNode struct {
	services    []runtime.Service
	cfg         *config
	powchainCfg *powchain.Web3ServiceConfig
	stateFeed   *event.Feed
}

// StateFeed implements statefeed.Notifier.
func (b *BeaconNode) StateFeed() *event.Feed {
	return b.stateFeed
}

type config struct {
	dataDir            string
	shouldClearDB      bool
	shouldForceClearDB bool
	mmapInitialSize    int
	wsCheckpointStr    string
}

func New(ctx context.Context, opts ...Option) (*BeaconNode, error) {
	bn := &BeaconNode{}

	params.BeaconConfig().InitializeForkSchedule()

	for _, opt := range opts {
		if err := opt(bn); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (b *BeaconNode) Start() error {
	attestationPool := attestations.NewPool()
	_ = attestationPool

	forkChoiceStore := protoarray.New(
		0, /* justified epoch */
		0, /* finalized epoch */
		params.BeaconConfig().ZeroHash,
	)
	_ = forkChoiceStore
	for i := len(b.services); i > 0; i-- {
		b.services[i].Start()
	}
	return nil
}

func (b *BeaconNode) startDB(ctx context.Context) error {
	dbPath := filepath.Join(b.cfg.dataDir, kv.BeaconNodeDbDirName)
	log.WithField("databasePath", dbPath).Info("Checking DB")
	beaconDB, err := db.NewDB(ctx, dbPath, &kv.Config{
		InitialMMapSize: b.cfg.mmapInitialSize,
	})
	if err != nil {
		return err
	}
	clearDBConfirmed := false
	if b.cfg.shouldClearDB && !b.cfg.shouldForceClearDB {
		actionText := "This will delete your beacon chain database stored in your data directory. " +
			"Your database backups will not be removed - do you want to proceed? (Y/N)"
		deniedText := "Database will not be deleted. No changes have been made."
		clearDBConfirmed, err = cmd.ConfirmAction(actionText, deniedText)
		if err != nil {
			return err
		}
	}
	if clearDBConfirmed || b.cfg.shouldForceClearDB {
		log.Warning("Removing database")
		if err := beaconDB.Close(); err != nil {
			return errors.Wrap(err, "could not close db prior to clearing")
		}
		if err := beaconDB.ClearDB(); err != nil {
			return errors.Wrap(err, "could not clear database")
		}
		beaconDB, err = db.NewDB(ctx, dbPath, &kv.Config{
			InitialMMapSize: b.cfg.mmapInitialSize,
		})
		if err != nil {
			return errors.Wrap(err, "could not create new database")
		}
	}
	return nil
}

func (b *BeaconNode) startP2P(ctx context.Context, beaconDB db.ReadOnlyDatabase, opts []p2p.Option) error {
	opts = append(
		opts,
		p2p.WithDatabase(beaconDB),
		p2p.WithStateNotifier(b),
	)
	svc, err := p2p.NewService(ctx, opts...)
	if err != nil {
		return nil
	}
	_ = svc
	return nil
}

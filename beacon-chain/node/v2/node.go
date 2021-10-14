package v2

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
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

type Opt func(*BeaconNode)

type BeaconNode struct {
	services    []runtime.Service
	cfg         *config
	p2pCfg      *p2p.Config
	powchainCfg *powchain.Web3ServiceConfig
}

type config struct {
	dataDir            string
	shouldClearDB      bool
	shouldForceClearDB bool
	mmapInitialSize    int
	wsCheckpointStr    string
}

func New(ctx context.Context, opts ...Opt) (*BeaconNode, error) {
	bn := &BeaconNode{}

	params.BeaconConfig().InitializeForkSchedule()

	for _, opt := range opts {
		opt(bn)
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

func (b *BeaconNode) startP2P(ctx context.Context, beaconDB db.Database) error {
	svc, err := p2p.NewService(ctx, &p2p.Config{
		NoDiscovery: b.p2pCfg.NoDiscovery,
		StaticPeers: b.p2pCfg.StaticPeers,
		//BootstrapNodeAddr: bootstrapNodeAddrs,
		RelayNodeAddr: b.p2pCfg.RelayNodeAddr,
		DataDir:       b.cfg.dataDir,
		//LocalIP:       cliCtx.String(cmd.P2PIP.Name),
		//HostAddress:   cliCtx.String(cmd.P2PHost.Name),
		//HostDNS:       cliCtx.String(cmd.P2PHostDNS.Name),
		//PrivateKey:    cliCtx.String(cmd.P2PPrivKey.Name),
		//MetaDataDir:   cliCtx.String(cmd.P2PMetadata.Name),
		//TCPPort:       cliCtx.Uint(cmd.P2PTCPPort.Name),
		//UDPPort:       cliCtx.Uint(cmd.P2PUDPPort.Name),
		//MaxPeers:      cliCtx.Uint(cmd.P2PMaxPeers.Name),
		//AllowListCIDR: cliCtx.String(cmd.P2PAllowList.Name),
		//DenyListCIDR:  slice.SplitCommaSeparated(cliCtx.StringSlice(cmd.P2PDenyList.Name)),
		//EnableUPnP:    cliCtx.Bool(cmd.EnableUPnPFlag.Name),
		//DisableDiscv5: cliCtx.Bool(flags.DisableDiscv5.Name),
		//StateNotifier: b,
		DB: beaconDB,
	})
	if err != nil {
		return nil
	}
	_ = svc
	return nil
}

//
//	if err := d.RunMigrations(b.ctx); err != nil {
//		return err
//	}
//
//	b.db = d
//
//	depositCache, err := depositcache.New()
//	if err != nil {
//		return errors.Wrap(err, "could not create deposit cache")
//	}
//
//	b.depositCache = depositCache
//
//	if cliCtx.IsSet(flags.GenesisStatePath.Name) {
//		r, err := os.Open(cliCtx.String(flags.GenesisStatePath.Name))
//		if err != nil {
//			return err
//		}
//		defer func() {
//			if err := r.Close(); err != nil {
//				log.WithError(err).Error("Failed to close genesis file")
//			}
//		}()
//		if err := b.db.LoadGenesis(b.ctx, r); err != nil {
//			if err == db.ErrExistingGenesisState {
//				return errors.New("Genesis state flag specified but a genesis state " +
//					"exists already. Run again with --clear-db and/or ensure you are using the " +
//					"appropriate testnet flag to load the given genesis state.")
//			}
//			return errors.Wrap(err, "could not load genesis from file")
//		}
//	}
//	if err := b.db.EnsureEmbeddedGenesis(b.ctx); err != nil {
//		return err
//	}
//	knownContract, err := b.db.DepositContractAddress(b.ctx)
//	if err != nil {
//		return err
//	}
//	addr := common.HexToAddress(depositAddress)
//	if len(knownContract) == 0 {
//		if err := b.db.SaveDepositContractAddress(b.ctx, addr); err != nil {
//			return errors.Wrap(err, "could not save deposit contract")
//		}
//	}
//	if len(knownContract) > 0 && !bytes.Equal(addr.Bytes(), knownContract) {
//		return fmt.Errorf("database contract is %#x but tried to run with %#x. This likely means "+
//			"you are trying to run on a different network than what the database contains. You can run once with "+
//			"'--clear-db' to wipe the old database or use an alternative data directory with '--datadir'",
//			knownContract, addr.Bytes())
//	}
//	log.Infof("Deposit contract: %#x", addr.Bytes())
//	return nil

// Package node is the main service which launches a beacon node and manages
// the lifecycle of all its associated services at runtime, such as p2p, RPC, sync,
// gracefully closing them if the process ends.
package node

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	apigateway "github.com/prysmaticlabs/prysm/api/gateway"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/slasherkv"
	interopcoldstart "github.com/prysmaticlabs/prysm/beacon-chain/deterministic-genesis"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	doublylinkedtree "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/gateway"
	"github.com/prysmaticlabs/prysm/beacon-chain/monitor"
	"github.com/prysmaticlabs/prysm/beacon-chain/node/registration"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/beacon-chain/slasher"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	regularsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/backfill"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/checkpoint"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/genesis"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/slice"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/backup"
	"github.com/prysmaticlabs/prysm/monitoring/prometheus"
	"github.com/prysmaticlabs/prysm/runtime"
	"github.com/prysmaticlabs/prysm/runtime/debug"
	"github.com/prysmaticlabs/prysm/runtime/prereqs"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const testSkipPowFlag = "test-skip-pow"

// 128MB max message size when enabling debug endpoints.
const debugGrpcMaxMsgSize = 1 << 27

// Used as a struct to keep cli flag options for configuring services
// for the beacon node. We keep this as a separate struct to not pollute the actual BeaconNode
// struct, as it is merely used to pass down configuration options into the appropriate services.
type serviceFlagOpts struct {
	blockchainFlagOpts []blockchain.Option
	powchainFlagOpts   []powchain.Option
}

// BeaconNode defines a struct that handles the services running a random beacon chain
// full PoS node. It handles the lifecycle of the entire system and registers
// services to a service registry.
type BeaconNode struct {
	cliCtx                  *cli.Context
	ctx                     context.Context
	cancel                  context.CancelFunc
	services                *runtime.ServiceRegistry
	lock                    sync.RWMutex
	stop                    chan struct{} // Channel to wait for termination notifications.
	db                      db.Database
	slasherDB               db.SlasherDatabase
	attestationPool         attestations.Pool
	exitPool                voluntaryexits.PoolManager
	slashingsPool           slashings.PoolManager
	syncCommitteePool       synccommittee.Pool
	depositCache            *depositcache.DepositCache
	stateFeed               *event.Feed
	blockFeed               *event.Feed
	opFeed                  *event.Feed
	forkChoiceStore         forkchoice.ForkChoicer
	stateGen                *stategen.State
	collector               *bcnodeCollector
	slasherBlockHeadersFeed *event.Feed
	slasherAttestationsFeed *event.Feed
	finalizedStateAtStartUp state.BeaconState
	serviceFlagOpts         *serviceFlagOpts
	blockchainFlagOpts      []blockchain.Option
	GenesisInitializer      genesis.Initializer
	CheckpointInitializer   checkpoint.Initializer
}

// New creates a new node instance, sets up configuration options, and registers
// every required service to the node.
func New(cliCtx *cli.Context, opts ...Option) (*BeaconNode, error) {
	if err := configureTracing(cliCtx); err != nil {
		return nil, err
	}
	prereqs.WarnIfPlatformNotSupported(cliCtx.Context)
	features.ConfigureBeaconChain(cliCtx)
	cmd.ConfigureBeaconChain(cliCtx)
	flags.ConfigureGlobalFlags(cliCtx)
	configureChainConfig(cliCtx)
	configureHistoricalSlasher(cliCtx)
	configureSafeSlotsToImportOptimistically(cliCtx)
	configureSlotsPerArchivedPoint(cliCtx)
	configureEth1Config(cliCtx)
	configureNetwork(cliCtx)
	configureInteropConfig(cliCtx)
	if err := configureExecutionSetting(cliCtx); err != nil {
		return nil, err
	}

	// Initializes any forks here.
	params.BeaconConfig().InitializeForkSchedule()

	registry := runtime.NewServiceRegistry()

	ctx, cancel := context.WithCancel(cliCtx.Context)
	beacon := &BeaconNode{
		cliCtx:                  cliCtx,
		ctx:                     ctx,
		cancel:                  cancel,
		services:                registry,
		stop:                    make(chan struct{}),
		stateFeed:               new(event.Feed),
		blockFeed:               new(event.Feed),
		opFeed:                  new(event.Feed),
		attestationPool:         attestations.NewPool(),
		exitPool:                voluntaryexits.NewPool(),
		slashingsPool:           slashings.NewPool(),
		syncCommitteePool:       synccommittee.NewPool(),
		slasherBlockHeadersFeed: new(event.Feed),
		slasherAttestationsFeed: new(event.Feed),
		serviceFlagOpts:         &serviceFlagOpts{},
	}

	for _, opt := range opts {
		if err := opt(beacon); err != nil {
			return nil, err
		}
	}

	depositAddress, err := powchain.DepositContractAddress()
	if err != nil {
		return nil, err
	}
	log.Debugln("Starting DB")
	if err := beacon.startDB(cliCtx, depositAddress); err != nil {
		return nil, err
	}

	log.Debugln("Starting Slashing DB")
	if err := beacon.startSlasherDB(cliCtx); err != nil {
		return nil, err
	}

	bfs := backfill.NewStatus(beacon.db)
	if err := bfs.Reload(ctx); err != nil {
		return nil, errors.Wrap(err, "backfill status initialization error")
	}

	log.Debugln("Starting State Gen")
	if err := beacon.startStateGen(ctx, bfs); err != nil {
		return nil, err
	}

	log.Debugln("Registering P2P Service")
	if err := beacon.registerP2P(cliCtx); err != nil {
		return nil, err
	}

	log.Debugln("Registering POW Chain Service")
	if err := beacon.registerPOWChainService(); err != nil {
		return nil, err
	}

	log.Debugln("Registering Attestation Pool Service")
	if err := beacon.registerAttestationPool(); err != nil {
		return nil, err
	}

	log.Debugln("Registering Determinstic Genesis Service")
	if err := beacon.registerDeterminsticGenesisService(); err != nil {
		return nil, err
	}

	log.Debugln("Starting Fork Choice")
	beacon.startForkChoice()

	log.Debugln("Registering Blockchain Service")
	if err := beacon.registerBlockchainService(); err != nil {
		return nil, err
	}

	log.Debugln("Registering Intial Sync Service")
	if err := beacon.registerInitialSyncService(); err != nil {
		return nil, err
	}

	log.Debugln("Registering Sync Service")
	if err := beacon.registerSyncService(); err != nil {
		return nil, err
	}

	log.Debugln("Registering Slasher Service")
	if err := beacon.registerSlasherService(); err != nil {
		return nil, err
	}

	log.Debugln("Registering RPC Service")
	if err := beacon.registerRPCService(); err != nil {
		return nil, err
	}

	log.Debugln("Registering GRPC Gateway Service")
	if err := beacon.registerGRPCGateway(); err != nil {
		return nil, err
	}

	log.Debugln("Registering Validator Monitoring Service")
	if err := beacon.registerValidatorMonitorService(); err != nil {
		return nil, err
	}

	if !cliCtx.Bool(cmd.DisableMonitoringFlag.Name) {
		log.Debugln("Registering Prometheus Service")
		if err := beacon.registerPrometheusService(cliCtx); err != nil {
			return nil, err
		}
	}

	// db.DatabasePath is the path to the containing directory
	// db.NewDBFilename expands that to the canonical full path using
	// the same construction as NewDB()
	c, err := newBeaconNodePromCollector(db.NewDBFilename(beacon.db.DatabasePath()))
	if err != nil {
		return nil, err
	}
	beacon.collector = c

	return beacon, nil
}

// StateFeed implements statefeed.Notifier.
func (b *BeaconNode) StateFeed() *event.Feed {
	return b.stateFeed
}

// BlockFeed implements blockfeed.Notifier.
func (b *BeaconNode) BlockFeed() *event.Feed {
	return b.blockFeed
}

// OperationFeed implements opfeed.Notifier.
func (b *BeaconNode) OperationFeed() *event.Feed {
	return b.opFeed
}

// Start the BeaconNode and kicks off every registered service.
func (b *BeaconNode) Start() {
	b.lock.Lock()

	log.WithFields(logrus.Fields{
		"version": version.Version(),
	}).Info("Starting beacon node")

	b.services.StartAll()

	stop := b.stop
	b.lock.Unlock()

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		debug.Exit(b.cliCtx) // Ensure trace and CPU profile data are flushed.
		go b.Close()
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
func (b *BeaconNode) Close() {
	b.lock.Lock()
	defer b.lock.Unlock()

	log.Info("Stopping beacon node")
	b.services.StopAll()
	if err := b.db.Close(); err != nil {
		log.Errorf("Failed to close database: %v", err)
	}
	b.collector.unregister()
	b.cancel()
	close(b.stop)
}

func (b *BeaconNode) startForkChoice() {
	if features.Get().EnableForkChoiceDoublyLinkedTree {
		b.forkChoiceStore = doublylinkedtree.New(0, 0)
	} else {
		b.forkChoiceStore = protoarray.New(0, 0, params.BeaconConfig().ZeroHash)
	}
}

func (b *BeaconNode) startDB(cliCtx *cli.Context, depositAddress string) error {
	baseDir := cliCtx.String(cmd.DataDirFlag.Name)
	dbPath := filepath.Join(baseDir, kv.BeaconNodeDbDirName)
	clearDB := cliCtx.Bool(cmd.ClearDB.Name)
	forceClearDB := cliCtx.Bool(cmd.ForceClearDB.Name)

	log.WithField("database-path", dbPath).Info("Checking DB")

	d, err := db.NewDB(b.ctx, dbPath, &kv.Config{
		InitialMMapSize: cliCtx.Int(cmd.BoltMMapInitialSizeFlag.Name),
	})
	if err != nil {
		return err
	}
	clearDBConfirmed := false
	if clearDB && !forceClearDB {
		actionText := "This will delete your beacon chain database stored in your data directory. " +
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
			return errors.Wrap(err, "could not clear database")
		}
		d, err = db.NewDB(b.ctx, dbPath, &kv.Config{
			InitialMMapSize: cliCtx.Int(cmd.BoltMMapInitialSizeFlag.Name),
		})
		if err != nil {
			return errors.Wrap(err, "could not create new database")
		}
	}

	if err := d.RunMigrations(b.ctx); err != nil {
		return err
	}

	b.db = d

	depositCache, err := depositcache.New()
	if err != nil {
		return errors.Wrap(err, "could not create deposit cache")
	}
	b.depositCache = depositCache

	if b.GenesisInitializer != nil {
		if err := b.GenesisInitializer.Initialize(b.ctx, d); err != nil {
			if err == db.ErrExistingGenesisState {
				return errors.New("Genesis state flag specified but a genesis state " +
					"exists already. Run again with --clear-db and/or ensure you are using the " +
					"appropriate testnet flag to load the given genesis state.")
			}
			return errors.Wrap(err, "could not load genesis from file")
		}
	}

	if err := b.db.EnsureEmbeddedGenesis(b.ctx); err != nil {
		return err
	}

	if b.CheckpointInitializer != nil {
		if err := b.CheckpointInitializer.Initialize(b.ctx, d); err != nil {
			return err
		}
	}

	knownContract, err := b.db.DepositContractAddress(b.ctx)
	if err != nil {
		return err
	}
	addr := common.HexToAddress(depositAddress)
	if len(knownContract) == 0 {
		if err := b.db.SaveDepositContractAddress(b.ctx, addr); err != nil {
			return errors.Wrap(err, "could not save deposit contract")
		}
	}
	if len(knownContract) > 0 && !bytes.Equal(addr.Bytes(), knownContract) {
		return fmt.Errorf("database contract is %#x but tried to run with %#x. This likely means "+
			"you are trying to run on a different network than what the database contains. You can run once with "+
			"'--clear-db' to wipe the old database or use an alternative data directory with '--datadir'",
			knownContract, addr.Bytes())
	}
	log.Infof("Deposit contract: %#x", addr.Bytes())
	return nil
}

func (b *BeaconNode) startSlasherDB(cliCtx *cli.Context) error {
	if !features.Get().EnableSlasher {
		return nil
	}
	baseDir := cliCtx.String(cmd.DataDirFlag.Name)
	dbPath := filepath.Join(baseDir, kv.BeaconNodeDbDirName)
	clearDB := cliCtx.Bool(cmd.ClearDB.Name)
	forceClearDB := cliCtx.Bool(cmd.ForceClearDB.Name)

	log.WithField("database-path", dbPath).Info("Checking DB")

	d, err := slasherkv.NewKVStore(b.ctx, dbPath, &slasherkv.Config{
		InitialMMapSize: cliCtx.Int(cmd.BoltMMapInitialSizeFlag.Name),
	})
	if err != nil {
		return err
	}
	clearDBConfirmed := false
	if clearDB && !forceClearDB {
		actionText := "This will delete your beacon chain database stored in your data directory. " +
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
			return errors.Wrap(err, "could not clear database")
		}
		d, err = slasherkv.NewKVStore(b.ctx, dbPath, &slasherkv.Config{
			InitialMMapSize: cliCtx.Int(cmd.BoltMMapInitialSizeFlag.Name),
		})
		if err != nil {
			return errors.Wrap(err, "could not create new database")
		}
	}

	b.slasherDB = d
	return nil
}

func (b *BeaconNode) startStateGen(ctx context.Context, bfs *backfill.Status) error {
	opts := []stategen.StateGenOption{stategen.WithBackfillStatus(bfs)}
	sg := stategen.New(b.db, opts...)

	cp, err := b.db.FinalizedCheckpoint(ctx)
	if err != nil {
		return err
	}

	r := bytesutil.ToBytes32(cp.Root)
	// Consider edge case where finalized root are zeros instead of genesis root hash.
	if r == params.BeaconConfig().ZeroHash {
		genesisBlock, err := b.db.GenesisBlock(ctx)
		if err != nil {
			return err
		}
		if genesisBlock != nil && !genesisBlock.IsNil() {
			r, err = genesisBlock.Block().HashTreeRoot()
			if err != nil {
				return err
			}
		}
	}

	b.finalizedStateAtStartUp, err = sg.StateByRoot(ctx, r)
	if err != nil {
		return err
	}

	b.stateGen = sg
	return nil
}

func (b *BeaconNode) registerP2P(cliCtx *cli.Context) error {
	bootstrapNodeAddrs, dataDir, err := registration.P2PPreregistration(cliCtx)
	if err != nil {
		return err
	}

	svc, err := p2p.NewService(b.ctx, &p2p.Config{
		NoDiscovery:       cliCtx.Bool(cmd.NoDiscovery.Name),
		StaticPeers:       slice.SplitCommaSeparated(cliCtx.StringSlice(cmd.StaticPeers.Name)),
		BootstrapNodeAddr: bootstrapNodeAddrs,
		RelayNodeAddr:     cliCtx.String(cmd.RelayNode.Name),
		DataDir:           dataDir,
		LocalIP:           cliCtx.String(cmd.P2PIP.Name),
		HostAddress:       cliCtx.String(cmd.P2PHost.Name),
		HostDNS:           cliCtx.String(cmd.P2PHostDNS.Name),
		PrivateKey:        cliCtx.String(cmd.P2PPrivKey.Name),
		MetaDataDir:       cliCtx.String(cmd.P2PMetadata.Name),
		TCPPort:           cliCtx.Uint(cmd.P2PTCPPort.Name),
		UDPPort:           cliCtx.Uint(cmd.P2PUDPPort.Name),
		MaxPeers:          cliCtx.Uint(cmd.P2PMaxPeers.Name),
		AllowListCIDR:     cliCtx.String(cmd.P2PAllowList.Name),
		DenyListCIDR:      slice.SplitCommaSeparated(cliCtx.StringSlice(cmd.P2PDenyList.Name)),
		EnableUPnP:        cliCtx.Bool(cmd.EnableUPnPFlag.Name),
		DisableDiscv5:     cliCtx.Bool(flags.DisableDiscv5.Name),
		StateNotifier:     b,
		DB:                b.db,
	})
	if err != nil {
		return err
	}
	return b.services.RegisterService(svc)
}

func (b *BeaconNode) fetchP2P() p2p.P2P {
	var p *p2p.Service
	if err := b.services.FetchService(&p); err != nil {
		panic(err)
	}
	return p
}

func (b *BeaconNode) registerAttestationPool() error {
	s, err := attestations.NewService(b.ctx, &attestations.Config{
		Pool: b.attestationPool,
	})
	if err != nil {
		return errors.Wrap(err, "could not register atts pool service")
	}
	return b.services.RegisterService(s)
}

func (b *BeaconNode) registerBlockchainService() error {
	var web3Service *powchain.Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	var attService *attestations.Service
	if err := b.services.FetchService(&attService); err != nil {
		return err
	}

	// skipcq: CRT-D0001
	opts := append(
		b.serviceFlagOpts.blockchainFlagOpts,
		blockchain.WithDatabase(b.db),
		blockchain.WithDepositCache(b.depositCache),
		blockchain.WithChainStartFetcher(web3Service),
		blockchain.WithExecutionEngineCaller(web3Service),
		blockchain.WithAttestationPool(b.attestationPool),
		blockchain.WithExitPool(b.exitPool),
		blockchain.WithSlashingPool(b.slashingsPool),
		blockchain.WithP2PBroadcaster(b.fetchP2P()),
		blockchain.WithStateNotifier(b),
		blockchain.WithForkChoiceStore(b.forkChoiceStore),
		blockchain.WithAttestationService(attService),
		blockchain.WithStateGen(b.stateGen),
		blockchain.WithSlasherAttestationsFeed(b.slasherAttestationsFeed),
		blockchain.WithFinalizedStateAtStartUp(b.finalizedStateAtStartUp),
	)
	blockchainService, err := blockchain.NewService(b.ctx, opts...)
	if err != nil {
		return errors.Wrap(err, "could not register blockchain service")
	}
	return b.services.RegisterService(blockchainService)
}

func (b *BeaconNode) registerPOWChainService() error {
	if b.cliCtx.Bool(testSkipPowFlag) {
		return b.services.RegisterService(&powchain.Service{})
	}
	bs, err := powchain.NewPowchainCollector(b.ctx)
	if err != nil {
		return err
	}
	depositContractAddr, err := powchain.DepositContractAddress()
	if err != nil {
		return err
	}

	// skipcq: CRT-D0001
	opts := append(
		b.serviceFlagOpts.powchainFlagOpts,
		powchain.WithDepositContractAddress(common.HexToAddress(depositContractAddr)),
		powchain.WithDatabase(b.db),
		powchain.WithDepositCache(b.depositCache),
		powchain.WithStateNotifier(b),
		powchain.WithStateGen(b.stateGen),
		powchain.WithBeaconNodeStatsUpdater(bs),
		powchain.WithFinalizedStateAtStartup(b.finalizedStateAtStartUp),
	)
	web3Service, err := powchain.NewService(b.ctx, opts...)
	if err != nil {
		return errors.Wrap(err, "could not register proof-of-work chain web3Service")
	}

	return b.services.RegisterService(web3Service)
}

func (b *BeaconNode) registerSyncService() error {
	var web3Service *powchain.Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var initSync *initialsync.Service
	if err := b.services.FetchService(&initSync); err != nil {
		return err
	}

	rs := regularsync.NewService(
		b.ctx,
		regularsync.WithDatabase(b.db),
		regularsync.WithP2P(b.fetchP2P()),
		regularsync.WithChainService(chainService),
		regularsync.WithInitialSync(initSync),
		regularsync.WithStateNotifier(b),
		regularsync.WithBlockNotifier(b),
		regularsync.WithAttestationNotifier(b),
		regularsync.WithOperationNotifier(b),
		regularsync.WithAttestationPool(b.attestationPool),
		regularsync.WithExitPool(b.exitPool),
		regularsync.WithSlashingPool(b.slashingsPool),
		regularsync.WithSyncCommsPool(b.syncCommitteePool),
		regularsync.WithStateGen(b.stateGen),
		regularsync.WithSlasherAttestationsFeed(b.slasherAttestationsFeed),
		regularsync.WithSlasherBlockHeadersFeed(b.slasherBlockHeadersFeed),
	)
	return b.services.RegisterService(rs)
}

func (b *BeaconNode) registerInitialSyncService() error {
	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	is := initialsync.NewService(b.ctx, &initialsync.Config{
		DB:            b.db,
		Chain:         chainService,
		P2P:           b.fetchP2P(),
		StateNotifier: b,
		BlockNotifier: b,
	})
	return b.services.RegisterService(is)
}

func (b *BeaconNode) registerSlasherService() error {
	if !features.Get().EnableSlasher {
		return nil
	}
	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}
	var syncService *initialsync.Service
	if err := b.services.FetchService(&syncService); err != nil {
		return err
	}

	slasherSrv, err := slasher.New(b.ctx, &slasher.ServiceConfig{
		IndexedAttestationsFeed: b.slasherAttestationsFeed,
		BeaconBlockHeadersFeed:  b.slasherBlockHeadersFeed,
		Database:                b.slasherDB,
		StateNotifier:           b,
		AttestationStateFetcher: chainService,
		StateGen:                b.stateGen,
		SlashingPoolInserter:    b.slashingsPool,
		SyncChecker:             syncService,
		HeadStateFetcher:        chainService,
	})
	if err != nil {
		return err
	}
	return b.services.RegisterService(slasherSrv)
}

func (b *BeaconNode) registerRPCService() error {
	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var web3Service *powchain.Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	var syncService *initialsync.Service
	if err := b.services.FetchService(&syncService); err != nil {
		return err
	}

	var slasherService *slasher.Service
	if features.Get().EnableSlasher {
		if err := b.services.FetchService(&slasherService); err != nil {
			return err
		}
	}

	genesisValidators := b.cliCtx.Uint64(flags.InteropNumValidatorsFlag.Name)
	genesisStatePath := b.cliCtx.String(flags.InteropGenesisStateFlag.Name)
	var depositFetcher depositcache.DepositFetcher
	var chainStartFetcher powchain.ChainStartFetcher
	if genesisValidators > 0 || genesisStatePath != "" {
		var interopService *interopcoldstart.Service
		if err := b.services.FetchService(&interopService); err != nil {
			return err
		}
		depositFetcher = interopService
		chainStartFetcher = interopService
	} else {
		depositFetcher = b.depositCache
		chainStartFetcher = web3Service
	}

	host := b.cliCtx.String(flags.RPCHost.Name)
	port := b.cliCtx.String(flags.RPCPort.Name)
	beaconMonitoringHost := b.cliCtx.String(cmd.MonitoringHostFlag.Name)
	beaconMonitoringPort := b.cliCtx.Int(flags.MonitoringPortFlag.Name)
	cert := b.cliCtx.String(flags.CertFlag.Name)
	key := b.cliCtx.String(flags.KeyFlag.Name)
	mockEth1DataVotes := b.cliCtx.Bool(flags.InteropMockEth1DataVotesFlag.Name)

	maxMsgSize := b.cliCtx.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name)
	enableDebugRPCEndpoints := b.cliCtx.Bool(flags.EnableDebugRPCEndpoints.Name)
	if enableDebugRPCEndpoints {
		maxMsgSize = int(math.Max(float64(maxMsgSize), debugGrpcMaxMsgSize))
	}

	p2pService := b.fetchP2P()
	rpcService := rpc.NewService(b.ctx, &rpc.Config{
		Host:                    host,
		Port:                    port,
		BeaconMonitoringHost:    beaconMonitoringHost,
		BeaconMonitoringPort:    beaconMonitoringPort,
		CertFlag:                cert,
		KeyFlag:                 key,
		BeaconDB:                b.db,
		Broadcaster:             p2pService,
		PeersFetcher:            p2pService,
		PeerManager:             p2pService,
		MetadataProvider:        p2pService,
		ChainInfoFetcher:        chainService,
		HeadFetcher:             chainService,
		CanonicalFetcher:        chainService,
		ForkFetcher:             chainService,
		FinalizationFetcher:     chainService,
		BlockReceiver:           chainService,
		AttestationReceiver:     chainService,
		GenesisTimeFetcher:      chainService,
		GenesisFetcher:          chainService,
		AttestationsPool:        b.attestationPool,
		ExitPool:                b.exitPool,
		SlashingsPool:           b.slashingsPool,
		SlashingChecker:         slasherService,
		SyncCommitteeObjectPool: b.syncCommitteePool,
		POWChainService:         web3Service,
		POWChainInfoFetcher:     web3Service,
		ChainStartFetcher:       chainStartFetcher,
		MockEth1Votes:           mockEth1DataVotes,
		SyncService:             syncService,
		DepositFetcher:          depositFetcher,
		PendingDepositFetcher:   b.depositCache,
		BlockNotifier:           b,
		StateNotifier:           b,
		OperationNotifier:       b,
		StateGen:                b.stateGen,
		EnableDebugRPCEndpoints: enableDebugRPCEndpoints,
		MaxMsgSize:              maxMsgSize,
		ExecutionEngineCaller:   web3Service,
	})

	return b.services.RegisterService(rpcService)
}

func (b *BeaconNode) registerPrometheusService(cliCtx *cli.Context) error {
	var additionalHandlers []prometheus.Handler
	var p *p2p.Service
	if err := b.services.FetchService(&p); err != nil {
		panic(err)
	}
	additionalHandlers = append(additionalHandlers, prometheus.Handler{Path: "/p2p", Handler: p.InfoHandler})

	var c *blockchain.Service
	if err := b.services.FetchService(&c); err != nil {
		panic(err)
	}

	if cliCtx.IsSet(cmd.EnableBackupWebhookFlag.Name) {
		additionalHandlers = append(
			additionalHandlers,
			prometheus.Handler{
				Path:    "/db/backup",
				Handler: backup.BackupHandler(b.db, cliCtx.String(cmd.BackupWebhookOutputDir.Name)),
			},
		)
	}

	service := prometheus.NewService(
		fmt.Sprintf("%s:%d", b.cliCtx.String(cmd.MonitoringHostFlag.Name), b.cliCtx.Int(flags.MonitoringPortFlag.Name)),
		b.services,
		additionalHandlers...,
	)
	hook := prometheus.NewLogrusCollector()
	logrus.AddHook(hook)
	return b.services.RegisterService(service)
}

func (b *BeaconNode) registerGRPCGateway() error {
	if b.cliCtx.Bool(flags.DisableGRPCGateway.Name) {
		return nil
	}
	gatewayPort := b.cliCtx.Int(flags.GRPCGatewayPort.Name)
	gatewayHost := b.cliCtx.String(flags.GRPCGatewayHost.Name)
	rpcHost := b.cliCtx.String(flags.RPCHost.Name)
	selfAddress := fmt.Sprintf("%s:%d", rpcHost, b.cliCtx.Int(flags.RPCPort.Name))
	gatewayAddress := fmt.Sprintf("%s:%d", gatewayHost, gatewayPort)
	allowedOrigins := strings.Split(b.cliCtx.String(flags.GPRCGatewayCorsDomain.Name), ",")
	enableDebugRPCEndpoints := b.cliCtx.Bool(flags.EnableDebugRPCEndpoints.Name)
	selfCert := b.cliCtx.String(flags.CertFlag.Name)
	maxCallSize := b.cliCtx.Uint64(cmd.GrpcMaxCallRecvMsgSizeFlag.Name)
	httpModules := b.cliCtx.String(flags.HTTPModules.Name)
	timeout := b.cliCtx.Int(cmd.ApiTimeoutFlag.Name)
	if enableDebugRPCEndpoints {
		maxCallSize = uint64(math.Max(float64(maxCallSize), debugGrpcMaxMsgSize))
	}

	gatewayConfig := gateway.DefaultConfig(enableDebugRPCEndpoints, httpModules)
	muxs := make([]*apigateway.PbMux, 0)
	if gatewayConfig.V1AlphaPbMux != nil {
		muxs = append(muxs, gatewayConfig.V1AlphaPbMux)
	}
	if gatewayConfig.EthPbMux != nil {
		muxs = append(muxs, gatewayConfig.EthPbMux)
	}

	opts := []apigateway.Option{
		apigateway.WithGatewayAddr(gatewayAddress),
		apigateway.WithRemoteAddr(selfAddress),
		apigateway.WithPbHandlers(muxs),
		apigateway.WithMuxHandler(gatewayConfig.Handler),
		apigateway.WithRemoteCert(selfCert),
		apigateway.WithMaxCallRecvMsgSize(maxCallSize),
		apigateway.WithAllowedOrigins(allowedOrigins),
		apigateway.WithTimeout(uint64(timeout)),
	}
	if flags.EnableHTTPEthAPI(httpModules) {
		opts = append(opts, apigateway.WithApiMiddleware(&apimiddleware.BeaconEndpointFactory{}))
	}
	g, err := apigateway.New(b.ctx, opts...)
	if err != nil {
		return err
	}
	return b.services.RegisterService(g)
}

func (b *BeaconNode) registerDeterminsticGenesisService() error {
	genesisTime := b.cliCtx.Uint64(flags.InteropGenesisTimeFlag.Name)
	genesisValidators := b.cliCtx.Uint64(flags.InteropNumValidatorsFlag.Name)
	genesisStatePath := b.cliCtx.String(flags.InteropGenesisStateFlag.Name)

	if genesisValidators > 0 || genesisStatePath != "" {
		svc := interopcoldstart.NewService(b.ctx, &interopcoldstart.Config{
			GenesisTime:   genesisTime,
			NumValidators: genesisValidators,
			BeaconDB:      b.db,
			DepositCache:  b.depositCache,
			GenesisPath:   genesisStatePath,
		})

		// Register genesis state as start-up state when interop mode.
		// The start-up state gets reused across services.
		st, err := b.db.GenesisState(b.ctx)
		if err != nil {
			return err
		}
		b.finalizedStateAtStartUp = st

		return b.services.RegisterService(svc)
	}
	return nil
}

func (b *BeaconNode) registerValidatorMonitorService() error {
	if cmd.ValidatorMonitorIndicesFlag.Value == nil {
		return nil
	}
	cliSlice := cmd.ValidatorMonitorIndicesFlag.Value.Value()
	if cliSlice == nil {
		return nil
	}
	tracked := make([]types.ValidatorIndex, len(cliSlice))
	for i := range tracked {
		tracked[i] = types.ValidatorIndex(cliSlice[i])
	}

	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}
	monitorConfig := &monitor.ValidatorMonitorConfig{
		StateNotifier:       b,
		AttestationNotifier: b,
		StateGen:            b.stateGen,
		HeadFetcher:         chainService,
	}
	svc, err := monitor.NewService(b.ctx, monitorConfig, tracked)
	if err != nil {
		return err
	}
	return b.services.RegisterService(svc)
}

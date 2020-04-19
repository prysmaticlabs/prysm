// Package node defines the services that a beacon chain node would perform.
package node

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/archiver"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/gateway"
	interopcoldstart "github.com/prysmaticlabs/prysm/beacon-chain/interop-cold-start"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	prysmsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
	initialsyncold "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync-old"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

var log = logrus.WithField("prefix", "node")

const beaconChainDBName = "beaconchaindata"
const testSkipPowFlag = "test-skip-pow"

// BeaconNode defines a struct that handles the services running a random beacon chain
// full PoS node. It handles the lifecycle of the entire system and registers
// services to a service registry.
type BeaconNode struct {
	ctx               *cli.Context
	services          *shared.ServiceRegistry
	lock              sync.RWMutex
	stop              chan struct{} // Channel to wait for termination notifications.
	db                db.Database
	stateSummaryCache *cache.StateSummaryCache
	attestationPool   attestations.Pool
	exitPool          *voluntaryexits.Pool
	slashingsPool     *slashings.Pool
	depositCache      *depositcache.DepositCache
	stateFeed         *event.Feed
	blockFeed         *event.Feed
	opFeed            *event.Feed
	forkChoiceStore   forkchoice.ForkChoicer
	stateGen          *stategen.State
}

// NewBeaconNode creates a new node instance, sets up configuration options, and registers
// every required service to the node.
func NewBeaconNode(ctx *cli.Context) (*BeaconNode, error) {
	if err := tracing.Setup(
		"beacon-chain", // service name
		ctx.String(cmd.TracingProcessNameFlag.Name),
		ctx.String(cmd.TracingEndpointFlag.Name),
		ctx.Float64(cmd.TraceSampleFractionFlag.Name),
		ctx.Bool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}

	featureconfig.ConfigureBeaconChain(ctx)
	flags.ConfigureGlobalFlags(ctx)
	registry := shared.NewServiceRegistry()

	beacon := &BeaconNode{
		ctx:               ctx,
		services:          registry,
		stop:              make(chan struct{}),
		stateFeed:         new(event.Feed),
		blockFeed:         new(event.Feed),
		opFeed:            new(event.Feed),
		attestationPool:   attestations.NewPool(),
		exitPool:          voluntaryexits.NewPool(),
		slashingsPool:     slashings.NewPool(),
		stateSummaryCache: cache.NewStateSummaryCache(),
	}

	if err := beacon.startDB(ctx); err != nil {
		return nil, err
	}

	beacon.startStateGen()

	if err := beacon.registerP2P(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerPOWChainService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerAttestationPool(); err != nil {
		return nil, err
	}

	if err := beacon.registerInteropServices(ctx); err != nil {
		return nil, err
	}

	beacon.startForkChoice()

	if err := beacon.registerBlockchainService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerInitialSyncService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerSyncService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerRPCService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerGRPCGateway(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerArchiverService(ctx); err != nil {
		return nil, err
	}

	if !ctx.Bool(cmd.DisableMonitoringFlag.Name) {
		if err := beacon.registerPrometheusService(ctx); err != nil {
			return nil, err
		}
	}

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
		"version": version.GetVersion(),
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
		debug.Exit(b.ctx) // Ensure trace and CPU profile data are flushed.
		go b.Close()
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
func (b *BeaconNode) Close() {
	b.lock.Lock()
	defer b.lock.Unlock()

	log.Info("Stopping beacon node")
	b.services.StopAll()
	if err := b.db.Close(); err != nil {
		log.Errorf("Failed to close database: %v", err)
	}
	close(b.stop)
}

func (b *BeaconNode) startForkChoice() {
	f := protoarray.New(0, 0, params.BeaconConfig().ZeroHash)
	b.forkChoiceStore = f
}

func (b *BeaconNode) startDB(ctx *cli.Context) error {
	baseDir := ctx.String(cmd.DataDirFlag.Name)
	dbPath := path.Join(baseDir, beaconChainDBName)
	clearDB := ctx.Bool(cmd.ClearDB.Name)
	forceClearDB := ctx.Bool(cmd.ForceClearDB.Name)

	d, err := db.NewDB(dbPath, b.stateSummaryCache)
	if err != nil {
		return err
	}
	clearDBConfirmed := false
	if clearDB && !forceClearDB {
		actionText := "This will delete your beacon chain data base stored in your data directory. " +
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
		d, err = db.NewDB(dbPath, b.stateSummaryCache)
		if err != nil {
			return err
		}
	} else {
		if err := d.HistoricalStatesDeleted(ctx); err != nil {
			return err
		}
	}

	log.WithField("database-path", dbPath).Info("Checking DB")
	b.db = d
	b.depositCache = depositcache.NewDepositCache()
	return nil
}

func (b *BeaconNode) startStateGen() {
	b.stateGen = stategen.New(b.db, b.stateSummaryCache)
}

func (b *BeaconNode) registerP2P(ctx *cli.Context) error {
	// Bootnode ENR may be a filepath to an ENR file.
	bootnodeAddrs := strings.Split(ctx.String(cmd.BootstrapNode.Name), ",")
	for i, addr := range bootnodeAddrs {
		if filepath.Ext(addr) == ".enr" {
			b, err := ioutil.ReadFile(addr)
			if err != nil {
				return err
			}
			bootnodeAddrs[i] = string(b)
		}
	}

	datadir := ctx.String(cmd.DataDirFlag.Name)
	if datadir == "" {
		datadir = cmd.DefaultDataDir()
	}

	svc, err := p2p.NewService(&p2p.Config{
		NoDiscovery:       ctx.Bool(cmd.NoDiscovery.Name),
		StaticPeers:       sliceutil.SplitCommaSeparated(ctx.StringSlice(cmd.StaticPeers.Name)),
		BootstrapNodeAddr: bootnodeAddrs,
		RelayNodeAddr:     ctx.String(cmd.RelayNode.Name),
		DataDir:           datadir,
		LocalIP:           ctx.String(cmd.P2PIP.Name),
		HostAddress:       ctx.String(cmd.P2PHost.Name),
		HostDNS:           ctx.String(cmd.P2PHostDNS.Name),
		PrivateKey:        ctx.String(cmd.P2PPrivKey.Name),
		MetaDataDir:       ctx.String(cmd.P2PMetadata.Name),
		TCPPort:           ctx.Uint(cmd.P2PTCPPort.Name),
		UDPPort:           ctx.Uint(cmd.P2PUDPPort.Name),
		MaxPeers:          ctx.Uint(cmd.P2PMaxPeers.Name),
		WhitelistCIDR:     ctx.String(cmd.P2PWhitelist.Name),
		EnableUPnP:        ctx.Bool(cmd.EnableUPnPFlag.Name),
		DisableDiscv5:     ctx.Bool(flags.DisableDiscv5.Name),
		Encoding:          ctx.String(cmd.P2PEncoding.Name),
		StateNotifier:     b,
		PubSub:            ctx.String(cmd.P2PPubsub.Name),
	})
	if err != nil {
		return err
	}
	return b.services.RegisterService(svc)
}

func (b *BeaconNode) fetchP2P(ctx *cli.Context) p2p.P2P {
	var p *p2p.Service
	if err := b.services.FetchService(&p); err != nil {
		panic(err)
	}
	return p
}

func (b *BeaconNode) registerAttestationPool() error {
	s, err := attestations.NewService(context.Background(), &attestations.Config{
		Pool: b.attestationPool,
	})
	if err != nil {
		return errors.Wrap(err, "could not register atts pool service")
	}
	return b.services.RegisterService(s)
}

func (b *BeaconNode) registerBlockchainService(ctx *cli.Context) error {
	var web3Service *powchain.Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	var opsService *attestations.Service
	if err := b.services.FetchService(&opsService); err != nil {
		return err
	}

	maxRoutines := ctx.Int64(cmd.MaxGoroutines.Name)
	blockchainService, err := blockchain.NewService(context.Background(), &blockchain.Config{
		BeaconDB:          b.db,
		DepositCache:      b.depositCache,
		ChainStartFetcher: web3Service,
		AttPool:           b.attestationPool,
		ExitPool:          b.exitPool,
		SlashingPool:      b.slashingsPool,
		P2p:               b.fetchP2P(ctx),
		MaxRoutines:       maxRoutines,
		StateNotifier:     b,
		ForkChoiceStore:   b.forkChoiceStore,
		OpsService:        opsService,
		StateGen:          b.stateGen,
	})
	if err != nil {
		return errors.Wrap(err, "could not register blockchain service")
	}
	return b.services.RegisterService(blockchainService)
}

func (b *BeaconNode) registerPOWChainService(cliCtx *cli.Context) error {
	if cliCtx.Bool(testSkipPowFlag) {
		return b.services.RegisterService(&powchain.Service{})
	}
	depAddress := cliCtx.String(flags.DepositContractFlag.Name)
	if depAddress == "" {
		log.Fatal(fmt.Sprintf("%s is required", flags.DepositContractFlag.Name))
	}

	if !common.IsHexAddress(depAddress) {
		log.Fatalf("Invalid deposit contract address given: %s", depAddress)
	}

	ctx := context.Background()
	cfg := &powchain.Web3ServiceConfig{
		ETH1Endpoint:    cliCtx.String(flags.Web3ProviderFlag.Name),
		HTTPEndPoint:    cliCtx.String(flags.HTTPWeb3ProviderFlag.Name),
		DepositContract: common.HexToAddress(depAddress),
		BeaconDB:        b.db,
		DepositCache:    b.depositCache,
		StateNotifier:   b,
	}
	web3Service, err := powchain.NewService(ctx, cfg)
	if err != nil {
		return errors.Wrap(err, "could not register proof-of-work chain web3Service")
	}
	knownContract, err := b.db.DepositContractAddress(ctx)
	if err != nil {
		return err
	}
	if len(knownContract) == 0 {
		if err := b.db.SaveDepositContractAddress(ctx, cfg.DepositContract); err != nil {
			return errors.Wrap(err, "could not save deposit contract")
		}
	}
	if len(knownContract) > 0 && !bytes.Equal(cfg.DepositContract.Bytes(), knownContract) {
		return fmt.Errorf("database contract is %#x but tried to run with %#x", knownContract, cfg.DepositContract.Bytes())
	}
	return b.services.RegisterService(web3Service)
}

func (b *BeaconNode) registerSyncService(ctx *cli.Context) error {
	var web3Service *powchain.Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var initSync prysmsync.Checker
	if cfg := featureconfig.Get(); cfg.DisableInitSyncQueue {
		var initSyncTmp *initialsyncold.Service
		if err := b.services.FetchService(&initSyncTmp); err != nil {
			return err
		}
		initSync = initSyncTmp
	} else {
		var initSyncTmp *initialsync.Service
		if err := b.services.FetchService(&initSyncTmp); err != nil {
			return err
		}
		initSync = initSyncTmp
	}

	rs := prysmsync.NewRegularSync(&prysmsync.Config{
		DB:                  b.db,
		P2P:                 b.fetchP2P(ctx),
		Chain:               chainService,
		InitialSync:         initSync,
		StateNotifier:       b,
		BlockNotifier:       b,
		AttestationNotifier: b,
		AttPool:             b.attestationPool,
		ExitPool:            b.exitPool,
		SlashingPool:        b.slashingsPool,
		StateSummaryCache:   b.stateSummaryCache,
		StateGen:            b.stateGen,
	})

	return b.services.RegisterService(rs)
}

func (b *BeaconNode) registerInitialSyncService(ctx *cli.Context) error {
	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	if cfg := featureconfig.Get(); cfg.DisableInitSyncQueue {
		is := initialsyncold.NewInitialSync(&initialsyncold.Config{
			DB:            b.db,
			Chain:         chainService,
			P2P:           b.fetchP2P(ctx),
			StateNotifier: b,
			BlockNotifier: b,
		})
		return b.services.RegisterService(is)
	}

	is := initialsync.NewInitialSync(&initialsync.Config{
		DB:            b.db,
		Chain:         chainService,
		P2P:           b.fetchP2P(ctx),
		StateNotifier: b,
		BlockNotifier: b,
	})
	return b.services.RegisterService(is)
}

func (b *BeaconNode) registerRPCService(ctx *cli.Context) error {
	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var web3Service *powchain.Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	var syncService prysmsync.Checker
	if cfg := featureconfig.Get(); cfg.DisableInitSyncQueue {
		var initSyncTmp *initialsyncold.Service
		if err := b.services.FetchService(&initSyncTmp); err != nil {
			return err
		}
		syncService = initSyncTmp
	} else {
		var initSyncTmp *initialsync.Service
		if err := b.services.FetchService(&initSyncTmp); err != nil {
			return err
		}
		syncService = initSyncTmp
	}

	genesisValidators := ctx.Uint64(flags.InteropNumValidatorsFlag.Name)
	genesisStatePath := ctx.String(flags.InteropGenesisStateFlag.Name)
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

	host := ctx.String(flags.RPCHost.Name)
	port := ctx.String(flags.RPCPort.Name)
	cert := ctx.String(flags.CertFlag.Name)
	key := ctx.String(flags.KeyFlag.Name)
	slasherCert := ctx.String(flags.SlasherCertFlag.Name)
	slasherProvider := ctx.String(flags.SlasherProviderFlag.Name)

	mockEth1DataVotes := ctx.Bool(flags.InteropMockEth1DataVotesFlag.Name)
	rpcService := rpc.NewService(context.Background(), &rpc.Config{
		Host:                  host,
		Port:                  port,
		CertFlag:              cert,
		KeyFlag:               key,
		BeaconDB:              b.db,
		Broadcaster:           b.fetchP2P(ctx),
		PeersFetcher:          b.fetchP2P(ctx),
		HeadFetcher:           chainService,
		ForkFetcher:           chainService,
		FinalizationFetcher:   chainService,
		ParticipationFetcher:  chainService,
		BlockReceiver:         chainService,
		AttestationReceiver:   chainService,
		GenesisTimeFetcher:    chainService,
		AttestationsPool:      b.attestationPool,
		ExitPool:              b.exitPool,
		SlashingsPool:         b.slashingsPool,
		POWChainService:       web3Service,
		ChainStartFetcher:     chainStartFetcher,
		MockEth1Votes:         mockEth1DataVotes,
		SyncService:           syncService,
		DepositFetcher:        depositFetcher,
		PendingDepositFetcher: b.depositCache,
		BlockNotifier:         b,
		StateNotifier:         b,
		OperationNotifier:     b,
		SlasherCert:           slasherCert,
		SlasherProvider:       slasherProvider,
		StateGen:              b.stateGen,
	})

	return b.services.RegisterService(rpcService)
}

func (b *BeaconNode) registerPrometheusService(ctx *cli.Context) error {
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

	if featureconfig.Get().EnableBackupWebhook {
		additionalHandlers = append(additionalHandlers, prometheus.Handler{Path: "/db/backup", Handler: db.BackupHandler(b.db)})
	}

	additionalHandlers = append(additionalHandlers, prometheus.Handler{Path: "/tree", Handler: c.TreeHandler})

	service := prometheus.NewPrometheusService(
		fmt.Sprintf(":%d", ctx.Int64(flags.MonitoringPortFlag.Name)),
		b.services,
		additionalHandlers...,
	)
	hook := prometheus.NewLogrusCollector()
	logrus.AddHook(hook)
	return b.services.RegisterService(service)
}

func (b *BeaconNode) registerGRPCGateway(ctx *cli.Context) error {
	gatewayPort := ctx.Int(flags.GRPCGatewayPort.Name)
	if gatewayPort > 0 {
		selfAddress := fmt.Sprintf("127.0.0.1:%d", ctx.Int(flags.RPCPort.Name))
		gatewayAddress := fmt.Sprintf("0.0.0.0:%d", gatewayPort)
		allowedOrigins := strings.Split(ctx.String(flags.GPRCGatewayCorsDomain.Name), ",")
		return b.services.RegisterService(gateway.New(context.Background(), selfAddress, gatewayAddress, nil /*optional mux*/, allowedOrigins))
	}
	return nil
}

func (b *BeaconNode) registerInteropServices(ctx *cli.Context) error {
	genesisTime := ctx.Uint64(flags.InteropGenesisTimeFlag.Name)
	genesisValidators := ctx.Uint64(flags.InteropNumValidatorsFlag.Name)
	genesisStatePath := ctx.String(flags.InteropGenesisStateFlag.Name)

	if genesisValidators > 0 || genesisStatePath != "" {
		svc := interopcoldstart.NewColdStartService(context.Background(), &interopcoldstart.Config{
			GenesisTime:   genesisTime,
			NumValidators: genesisValidators,
			BeaconDB:      b.db,
			DepositCache:  b.depositCache,
			GenesisPath:   genesisStatePath,
		})

		return b.services.RegisterService(svc)
	}
	return nil
}

func (b *BeaconNode) registerArchiverService(ctx *cli.Context) error {
	if !flags.Get().EnableArchive {
		return nil
	}
	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}
	svc := archiver.NewArchiverService(context.Background(), &archiver.Config{
		BeaconDB:             b.db,
		HeadFetcher:          chainService,
		ParticipationFetcher: chainService,
		StateNotifier:        b,
	})
	return b.services.RegisterService(svc)
}

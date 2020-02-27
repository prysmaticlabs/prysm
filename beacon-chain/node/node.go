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
	prysmsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
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
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "node")

const beaconChainDBName = "beaconchaindata"
const testSkipPowFlag = "test-skip-pow"

// BeaconNode defines a struct that handles the services running a random beacon chain
// full PoS node. It handles the lifecycle of the entire system and registers
// services to a service registry.
type BeaconNode struct {
	ctx             *cli.Context
	services        *shared.ServiceRegistry
	lock            sync.RWMutex
	stop            chan struct{} // Channel to wait for termination notifications.
	db              db.Database
	attestationPool attestations.Pool
	exitPool        *voluntaryexits.Pool
	slashingsPool   *slashings.Pool
	depositCache    *depositcache.DepositCache
	stateFeed       *event.Feed
	blockFeed       *event.Feed
	opFeed          *event.Feed
	forkChoiceStore forkchoice.ForkChoicer
}

// NewBeaconNode creates a new node instance, sets up configuration options, and registers
// every required service to the node.
func NewBeaconNode(ctx *cli.Context) (*BeaconNode, error) {
	if err := tracing.Setup(
		"beacon-chain", // service name
		ctx.GlobalString(cmd.TracingProcessNameFlag.Name),
		ctx.GlobalString(cmd.TracingEndpointFlag.Name),
		ctx.GlobalFloat64(cmd.TraceSampleFractionFlag.Name),
		ctx.GlobalBool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}
	featureconfig.ConfigureBeaconChain(ctx)
	flags.ConfigureGlobalFlags(ctx)
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

	beacon := &BeaconNode{
		ctx:             ctx,
		services:        registry,
		stop:            make(chan struct{}),
		stateFeed:       new(event.Feed),
		blockFeed:       new(event.Feed),
		opFeed:          new(event.Feed),
		attestationPool: attestations.NewPool(),
		exitPool:        voluntaryexits.NewPool(),
		slashingsPool:   slashings.NewPool(),
	}

	if err := beacon.startDB(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerP2P(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerPOWChainService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerAttestationPool(ctx); err != nil {
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

	if !ctx.GlobalBool(cmd.DisableMonitoringFlag.Name) {
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
	baseDir := ctx.GlobalString(cmd.DataDirFlag.Name)
	dbPath := path.Join(baseDir, beaconChainDBName)
	clearDB := ctx.GlobalBool(cmd.ClearDB.Name)
	forceClearDB := ctx.GlobalBool(cmd.ForceClearDB.Name)

	d, err := db.NewDB(dbPath)
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
		d, err = db.NewDB(dbPath)
		if err != nil {
			return err
		}
	}
	log.WithField("database-path", dbPath).Info("Checking DB")
	b.db = d
	b.depositCache = depositcache.NewDepositCache()
	return nil
}

func (b *BeaconNode) registerP2P(ctx *cli.Context) error {
	// Bootnode ENR may be a filepath to an ENR file.
	bootnodeAddrs := strings.Split(ctx.GlobalString(cmd.BootstrapNode.Name), ",")
	for i, addr := range bootnodeAddrs {
		if filepath.Ext(addr) == ".enr" {
			b, err := ioutil.ReadFile(addr)
			if err != nil {
				return err
			}
			bootnodeAddrs[i] = string(b)
		}
	}

	svc, err := p2p.NewService(&p2p.Config{
		NoDiscovery:       ctx.GlobalBool(cmd.NoDiscovery.Name),
		StaticPeers:       sliceutil.SplitCommaSeparated(ctx.GlobalStringSlice(cmd.StaticPeers.Name)),
		BootstrapNodeAddr: bootnodeAddrs,
		RelayNodeAddr:     ctx.GlobalString(cmd.RelayNode.Name),
		DataDir:           ctx.GlobalString(cmd.DataDirFlag.Name),
		LocalIP:           ctx.GlobalString(cmd.P2PIP.Name),
		HostAddress:       ctx.GlobalString(cmd.P2PHost.Name),
		HostDNS:           ctx.GlobalString(cmd.P2PHostDNS.Name),
		PrivateKey:        ctx.GlobalString(cmd.P2PPrivKey.Name),
		TCPPort:           ctx.GlobalUint(cmd.P2PTCPPort.Name),
		UDPPort:           ctx.GlobalUint(cmd.P2PUDPPort.Name),
		MaxPeers:          ctx.GlobalUint(cmd.P2PMaxPeers.Name),
		WhitelistCIDR:     ctx.GlobalString(cmd.P2PWhitelist.Name),
		EnableUPnP:        ctx.GlobalBool(cmd.EnableUPnPFlag.Name),
		Encoding:          ctx.GlobalString(cmd.P2PEncoding.Name),
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

func (b *BeaconNode) registerBlockchainService(ctx *cli.Context) error {
	var web3Service *powchain.Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	maxRoutines := ctx.GlobalInt64(cmd.MaxGoroutines.Name)
	blockchainService, err := blockchain.NewService(context.Background(), &blockchain.Config{
		BeaconDB:          b.db,
		DepositCache:      b.depositCache,
		ChainStartFetcher: web3Service,
		AttPool:           b.attestationPool,
		ExitPool:          b.exitPool,
		P2p:               b.fetchP2P(ctx),
		MaxRoutines:       maxRoutines,
		StateNotifier:     b,
		ForkChoiceStore:   b.forkChoiceStore,
	})
	if err != nil {
		return errors.Wrap(err, "could not register blockchain service")
	}
	return b.services.RegisterService(blockchainService)
}

func (b *BeaconNode) registerAttestationPool(ctx *cli.Context) error {
	attPoolService, err := attestations.NewService(context.Background(), &attestations.Config{
		Pool: b.attestationPool,
	})
	if err != nil {
		return err
	}
	return b.services.RegisterService(attPoolService)
}

func (b *BeaconNode) registerPOWChainService(cliCtx *cli.Context) error {
	if cliCtx.GlobalBool(testSkipPowFlag) {
		return b.services.RegisterService(&powchain.Service{})
	}
	depAddress := cliCtx.GlobalString(flags.DepositContractFlag.Name)
	if depAddress == "" {
		log.Fatal(fmt.Sprintf("%s is required", flags.DepositContractFlag.Name))
	}

	if !common.IsHexAddress(depAddress) {
		log.Fatalf("Invalid deposit contract address given: %s", depAddress)
	}

	ctx := context.Background()
	cfg := &powchain.Web3ServiceConfig{
		ETH1Endpoint:    cliCtx.GlobalString(flags.Web3ProviderFlag.Name),
		HTTPEndPoint:    cliCtx.GlobalString(flags.HTTPWeb3ProviderFlag.Name),
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

	var initSync *initialsync.Service
	if err := b.services.FetchService(&initSync); err != nil {
		return err
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
	})

	return b.services.RegisterService(rs)
}

func (b *BeaconNode) registerInitialSyncService(ctx *cli.Context) error {
	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
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

	var syncService *initialsync.Service
	if err := b.services.FetchService(&syncService); err != nil {
		return err
	}

	genesisValidators := ctx.GlobalUint64(flags.InteropNumValidatorsFlag.Name)
	genesisStatePath := ctx.GlobalString(flags.InteropGenesisStateFlag.Name)
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

	host := ctx.GlobalString(flags.RPCHost.Name)
	port := ctx.GlobalString(flags.RPCPort.Name)
	cert := ctx.GlobalString(flags.CertFlag.Name)
	key := ctx.GlobalString(flags.KeyFlag.Name)
	slasherCert := ctx.GlobalString(flags.SlasherCertFlag.Name)
	slasherProvider := ctx.GlobalString(flags.SlasherProviderFlag.Name)

	mockEth1DataVotes := ctx.GlobalBool(flags.InteropMockEth1DataVotesFlag.Name)
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
		fmt.Sprintf(":%d", ctx.GlobalInt64(cmd.MonitoringPortFlag.Name)),
		b.services,
		additionalHandlers...,
	)
	hook := prometheus.NewLogrusCollector()
	logrus.AddHook(hook)
	return b.services.RegisterService(service)
}

func (b *BeaconNode) registerGRPCGateway(ctx *cli.Context) error {
	gatewayPort := ctx.GlobalInt(flags.GRPCGatewayPort.Name)
	if gatewayPort > 0 {
		selfAddress := fmt.Sprintf("127.0.0.1:%d", ctx.GlobalInt(flags.RPCPort.Name))
		gatewayAddress := fmt.Sprintf("0.0.0.0:%d", gatewayPort)
		return b.services.RegisterService(gateway.New(context.Background(), selfAddress, gatewayAddress, nil /*optional mux*/))
	}
	return nil
}

func (b *BeaconNode) registerInteropServices(ctx *cli.Context) error {
	genesisTime := ctx.GlobalUint64(flags.InteropGenesisTimeFlag.Name)
	genesisValidators := ctx.GlobalUint64(flags.InteropNumValidatorsFlag.Name)
	genesisStatePath := ctx.GlobalString(flags.InteropGenesisStateFlag.Name)

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

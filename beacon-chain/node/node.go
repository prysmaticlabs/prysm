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
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/archiver"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/gateway"
	interopcoldstart "github.com/prysmaticlabs/prysm/beacon-chain/interop-cold-start"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc"
	prysmsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
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
	ctx          *cli.Context
	services     *shared.ServiceRegistry
	lock         sync.RWMutex
	stop         chan struct{} // Channel to wait for termination notifications.
	db           db.Database
	depositCache *depositcache.DepositCache
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
	featureconfig.ConfigureBeaconFeatures(ctx)
	registry := shared.NewServiceRegistry()

	beacon := &BeaconNode{
		ctx:      ctx,
		services: registry,
		stop:     make(chan struct{}),
	}

	// Use custom config values if the --no-custom-config flag is set.
	if !ctx.GlobalBool(flags.NoCustomConfigFlag.Name) {
		log.Info("Using custom parameter configuration")
		if featureconfig.FeatureConfig().DemoConfig {
			log.Info("Using demo config")
			params.UseDemoBeaconConfig()
		} else {
			log.Info("Using minimal config")
			params.UseMinimalConfig()
		}
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

	if err := beacon.registerOperationService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerInteropServices(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerBlockchainService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerSyncService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerInitialSyncService(ctx); err != nil {
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

func (b *BeaconNode) startDB(ctx *cli.Context) error {
	baseDir := ctx.GlobalString(cmd.DataDirFlag.Name)
	dbPath := path.Join(baseDir, beaconChainDBName)
	d, err := db.NewDB(dbPath)
	if err != nil {
		return err
	}
	if b.ctx.GlobalBool(cmd.ClearDB.Name) {
		if err := d.ClearDB(); err != nil {
			return err
		}
		d, err = db.NewDB(dbPath)
		if err != nil {
			return err
		}
	}

	log.WithField("path", dbPath).Info("Checking db")
	b.db = d
	b.depositCache = depositcache.NewDepositCache()
	return nil
}

func (b *BeaconNode) registerP2P(ctx *cli.Context) error {
	// Bootnode ENR may be a filepath to an ENR file.
	bootnodeENR := ctx.GlobalString(cmd.BootstrapNode.Name)
	if filepath.Ext(bootnodeENR) == ".enr" {
		b, err := ioutil.ReadFile(bootnodeENR)
		if err != nil {
			return err
		}
		bootnodeENR = string(b)
	}

	svc, err := p2p.NewService(&p2p.Config{
		NoDiscovery:       ctx.GlobalBool(cmd.NoDiscovery.Name),
		StaticPeers:       sliceutil.SplitCommaSeparated(ctx.GlobalStringSlice(cmd.StaticPeers.Name)),
		BootstrapNodeAddr: bootnodeENR,
		RelayNodeAddr:     ctx.GlobalString(cmd.RelayNode.Name),
		DataDir:           ctx.GlobalString(cmd.DataDirFlag.Name),
		HostAddress:       ctx.GlobalString(cmd.P2PHost.Name),
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
	var opsService *operations.Service
	if err := b.services.FetchService(&opsService); err != nil {
		return err
	}

	maxRoutines := ctx.GlobalInt64(cmd.MaxGoroutines.Name)
	blockchainService, err := blockchain.NewService(context.Background(), &blockchain.Config{
		BeaconDB:          b.db,
		DepositCache:      b.depositCache,
		ChainStartFetcher: web3Service,
		OpsPoolService:    opsService,
		P2p:               b.fetchP2P(ctx),
		MaxRoutines:       maxRoutines,
	})
	if err != nil {
		return errors.Wrap(err, "could not register blockchain service")
	}
	return b.services.RegisterService(blockchainService)
}

func (b *BeaconNode) registerOperationService(ctx *cli.Context) error {
	operationService := operations.NewService(context.Background(), &operations.Config{
		BeaconDB: b.db,
	})

	return b.services.RegisterService(operationService)
}

func (b *BeaconNode) registerPOWChainService(cliCtx *cli.Context) error {
	if cliCtx.GlobalBool(testSkipPowFlag) {
		return b.services.RegisterService(&powchain.Service{})
	}
	depAddress := cliCtx.GlobalString(flags.DepositContractFlag.Name)
	if depAddress == "" {
		var err error
		depAddress, err = fetchDepositContract()
		if err != nil {
			log.WithError(err).Fatal("Cannot fetch deposit contract")
		}
	}

	if !common.IsHexAddress(depAddress) {
		log.Fatalf("Invalid deposit contract address given: %s", depAddress)
	}

	httpRPCClient, err := gethRPC.Dial(cliCtx.GlobalString(flags.HTTPWeb3ProviderFlag.Name))
	if err != nil {
		log.Fatalf("Access to PoW chain is required for validator. Unable to connect to Geth node: %v", err)
	}
	httpClient := ethclient.NewClient(httpRPCClient)

	rpcClient, err := gethRPC.Dial(cliCtx.GlobalString(flags.Web3ProviderFlag.Name))
	if err != nil {
		log.Fatalf("Access to PoW chain is required for validator. Unable to connect to Geth node: %v", err)
	}
	powClient := ethclient.NewClient(rpcClient)

	ctx := context.Background()
	cfg := &powchain.Web3ServiceConfig{
		Endpoint:        cliCtx.GlobalString(flags.Web3ProviderFlag.Name),
		DepositContract: common.HexToAddress(depAddress),
		Client:          httpClient,
		Reader:          powClient,
		Logger:          powClient,
		HTTPLogger:      httpClient,
		BlockFetcher:    httpClient,
		ContractBackend: httpClient,
		BeaconDB:        b.db,
		DepositCache:    b.depositCache,
	}
	web3Service, err := powchain.NewService(ctx, cfg)
	if err != nil {
		return errors.Wrap(err, "could not register proof-of-work chain web3Service")
	}
	knownContract, err := b.db.DepositContractAddress(ctx)
	if err != nil {
		return err
	}
	if len(knownContract) > 0 && !bytes.Equal(cfg.DepositContract.Bytes(), knownContract) {
		return fmt.Errorf("database contract is %#x but tried to run with %#x", knownContract, cfg.DepositContract.Bytes())
	}

	return b.services.RegisterService(web3Service)
}

func (b *BeaconNode) registerSyncService(ctx *cli.Context) error {
	var operationService *operations.Service
	if err := b.services.FetchService(&operationService); err != nil {
		return err
	}

	var web3Service *powchain.Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	rs := prysmsync.NewRegularSync(&prysmsync.Config{
		DB:         b.db,
		P2P:        b.fetchP2P(ctx),
		Operations: operationService,
		Chain:      chainService,
	})

	return b.services.RegisterService(rs)
}

func (b *BeaconNode) registerInitialSyncService(ctx *cli.Context) error {

	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var regSync *prysmsync.RegularSync
	if err := b.services.FetchService(&regSync); err != nil {
		return err
	}

	is := initialsync.NewInitialSync(&initialsync.Config{
		Chain:   chainService,
		RegSync: regSync,
		P2P:     b.fetchP2P(ctx),
	})

	return b.services.RegisterService(is)

}

func (b *BeaconNode) registerRPCService(ctx *cli.Context) error {
	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var operationService *operations.Service
	if err := b.services.FetchService(&operationService); err != nil {
		return err
	}

	var web3Service *powchain.Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	var syncService *prysmsync.RegularSync
	if err := b.services.FetchService(&syncService); err != nil {
		return err
	}

	genesisTime := ctx.GlobalUint64(flags.InteropGenesisTimeFlag.Name)
	genesisValidators := ctx.GlobalUint64(flags.InteropNumValidatorsFlag.Name)
	genesisStatePath := ctx.GlobalString(flags.InteropGenesisStateFlag.Name)
	var depositFetcher depositcache.DepositFetcher
	var chainStartFetcher powchain.ChainStartFetcher
	if genesisTime > 0 && genesisValidators > 0 || genesisStatePath != "" {
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

	port := ctx.GlobalString(flags.RPCPort.Name)
	cert := ctx.GlobalString(flags.CertFlag.Name)
	key := ctx.GlobalString(flags.KeyFlag.Name)
	mockEth1DataVotes := ctx.GlobalBool(flags.InteropMockEth1DataVotesFlag.Name)
	rpcService := rpc.NewService(context.Background(), &rpc.Config{
		Port:                  port,
		CertFlag:              cert,
		KeyFlag:               key,
		BeaconDB:              b.db,
		Broadcaster:           b.fetchP2P(ctx),
		HeadFetcher:           chainService,
		BlockReceiver:         chainService,
		AttestationReceiver:   chainService,
		StateFeedListener:     chainService,
		GenesisTimeFetcher:    chainService,
		AttestationsPool:      operationService,
		OperationsHandler:     operationService,
		POWChainService:       web3Service,
		ChainStartFetcher:     chainStartFetcher,
		MockEth1Votes:         mockEth1DataVotes,
		SyncService:           syncService,
		DepositFetcher:        depositFetcher,
		PendingDepositFetcher: b.depositCache,
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
	additionalHandlers = append(additionalHandlers, prometheus.Handler{Path: "/heads", Handler: c.HeadsHandler})

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
		gatewayAddress := fmt.Sprintf("127.0.0.1:%d", gatewayPort)
		return b.services.RegisterService(gateway.New(context.Background(), selfAddress, gatewayAddress, nil /*optional mux*/))
	}
	return nil
}

func (b *BeaconNode) registerInteropServices(ctx *cli.Context) error {
	genesisTime := ctx.GlobalUint64(flags.InteropGenesisTimeFlag.Name)
	genesisValidators := ctx.GlobalUint64(flags.InteropNumValidatorsFlag.Name)
	genesisStatePath := ctx.GlobalString(flags.InteropGenesisStateFlag.Name)

	if genesisTime > 0 && genesisValidators > 0 || genesisStatePath != "" {
		svc := interopcoldstart.NewColdStartService(context.Background(), &interopcoldstart.Config{
			GenesisTime:   genesisTime,
			NumValidators: genesisValidators,
			BeaconDB:      b.db,
			DepositCache:  b.depositCache,
			GenesisPath:   genesisStatePath,
		})

		return b.services.RegisterService(svc)
	} else if genesisTime+genesisValidators > 0 {
		log.Errorf("%s and %s must be used together", flags.InteropNumValidatorsFlag.Name, flags.InteropGenesisTimeFlag.Name)
	}
	return nil
}

func (b *BeaconNode) registerArchiverService(ctx *cli.Context) error {
	shouldArchive := ctx.GlobalBool(flags.ArchiveEnableFlag.Name)
	if !shouldArchive {
		return nil
	}
	var chainService *blockchain.Service
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}
	svc := archiver.NewArchiverService(context.Background(), &archiver.Config{
		BeaconDB:        b.db,
		NewHeadNotifier: chainService,
	})
	return b.services.RegisterService(svc)
}

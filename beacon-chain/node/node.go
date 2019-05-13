// Package node defines the services that a beacon chain node would perform.
package node

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/attestation"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc"
	rbcsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
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
	ctx      *cli.Context
	services *shared.ServiceRegistry
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications.
	db       *db.BeaconDB
}

// NewBeaconNode creates a new node instance, sets up configuration options, and registers
// every required service to the node.
func NewBeaconNode(ctx *cli.Context) (*BeaconNode, error) {
	if err := tracing.Setup(
		"beacon-chain", // service name
		ctx.GlobalString(cmd.TracingEndpointFlag.Name),
		ctx.GlobalFloat64(cmd.TraceSampleFractionFlag.Name),
		ctx.GlobalBool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}
	registry := shared.NewServiceRegistry()

	beacon := &BeaconNode{
		ctx:      ctx,
		services: registry,
		stop:     make(chan struct{}),
	}

	// Use custom config values if the --no-custom-config flag is set.
	if !ctx.GlobalBool(utils.NoCustomConfigFlag.Name) {
		log.Info("Using custom parameter configuration")
		params.UseDemoBeaconConfig()
	}

	featureconfig.ConfigureBeaconFeatures(ctx)

	if err := beacon.startDB(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerP2P(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerPOWChainService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerAttestationService(); err != nil {
		return nil, err
	}

	if err := beacon.registerOperationService(); err != nil {
		return nil, err
	}

	if err := beacon.registerBlockchainService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerSyncService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerRPCService(ctx); err != nil {
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
	if b.ctx.GlobalBool(cmd.ClearDB.Name) {
		if err := db.ClearDB(dbPath); err != nil {
			return err
		}
	}

	db, err := db.NewDB(dbPath)
	if err != nil {
		return err
	}

	log.WithField("path", dbPath).Info("Checking db")
	b.db = db
	return nil
}

func (b *BeaconNode) registerP2P(ctx *cli.Context) error {
	beaconp2p, err := configureP2P(ctx)
	if err != nil {
		return fmt.Errorf("could not register p2p service: %v", err)
	}

	return b.services.RegisterService(beaconp2p)
}

func (b *BeaconNode) registerBlockchainService(_ *cli.Context) error {
	var web3Service *powchain.Web3Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}
	var opsService *operations.Service
	if err := b.services.FetchService(&opsService); err != nil {
		return err
	}
	var attsService *attestation.Service
	if err := b.services.FetchService(&attsService); err != nil {
		return err
	}
	var p2pService *p2p.Server
	if err := b.services.FetchService(&p2pService); err != nil {
		return err
	}

	blockchainService, err := blockchain.NewChainService(context.Background(), &blockchain.Config{
		BeaconDB:       b.db,
		Web3Service:    web3Service,
		OpsPoolService: opsService,
		AttsService:    attsService,
		P2p:            p2pService,
	})
	if err != nil {
		return fmt.Errorf("could not register blockchain service: %v", err)
	}
	return b.services.RegisterService(blockchainService)
}

func (b *BeaconNode) registerOperationService() error {
	var p2pService *p2p.Server
	if err := b.services.FetchService(&p2pService); err != nil {
		return err
	}

	operationService := operations.NewOpsPoolService(context.Background(), &operations.Config{
		BeaconDB: b.db,
		P2P:      p2pService,
	})

	return b.services.RegisterService(operationService)
}

func (b *BeaconNode) registerPOWChainService(cliCtx *cli.Context) error {
	if cliCtx.GlobalBool(testSkipPowFlag) {
		return b.services.RegisterService(&powchain.Web3Service{})
	}

	depAddress := cliCtx.GlobalString(utils.DepositContractFlag.Name)

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

	rpcClient, err := gethRPC.Dial(cliCtx.GlobalString(utils.Web3ProviderFlag.Name))
	if err != nil {
		log.Fatalf("Access to PoW chain is required for validator. Unable to connect to Geth node: %v", err)
	}
	powClient := ethclient.NewClient(rpcClient)

	httpRPCClient, err := gethRPC.Dial(cliCtx.GlobalString(utils.HTTPWeb3ProviderFlag.Name))
	if err != nil {
		log.Fatalf("Access to PoW chain is required for validator. Unable to connect to Geth node: %v", err)
	}
	httpClient := ethclient.NewClient(httpRPCClient)

	ctx := context.Background()
	cfg := &powchain.Web3ServiceConfig{
		Endpoint:        cliCtx.GlobalString(utils.Web3ProviderFlag.Name),
		DepositContract: common.HexToAddress(depAddress),
		Client:          powClient,
		Reader:          powClient,
		Logger:          powClient,
		HTTPLogger:      httpClient,
		BlockFetcher:    httpClient,
		ContractBackend: powClient,
		BeaconDB:        b.db,
	}
	web3Service, err := powchain.NewWeb3Service(ctx, cfg)
	if err != nil {
		return fmt.Errorf("could not register proof-of-work chain web3Service: %v", err)
	}

	if err := b.db.VerifyContractAddress(ctx, cfg.DepositContract); err != nil {
		return err
	}

	return b.services.RegisterService(web3Service)
}

func (b *BeaconNode) registerSyncService(_ *cli.Context) error {
	var chainService *blockchain.ChainService
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var p2pService *p2p.Server
	if err := b.services.FetchService(&p2pService); err != nil {
		return err
	}

	var operationService *operations.Service
	if err := b.services.FetchService(&operationService); err != nil {
		return err
	}

	var attsService *attestation.Service
	if err := b.services.FetchService(&attsService); err != nil {
		return err
	}

	var web3Service *powchain.Web3Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	cfg := &rbcsync.Config{
		ChainService:     chainService,
		P2P:              p2pService,
		BeaconDB:         b.db,
		OperationService: operationService,
		PowChainService:  web3Service,
		AttsService:      attsService,
	}

	syncService := rbcsync.NewSyncService(context.Background(), cfg)
	return b.services.RegisterService(syncService)
}

func (b *BeaconNode) registerRPCService(ctx *cli.Context) error {
	var chainService *blockchain.ChainService
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var p2pService *p2p.Server
	if err := b.services.FetchService(&p2pService); err != nil {
		return err
	}

	var operationService *operations.Service
	if err := b.services.FetchService(&operationService); err != nil {
		return err
	}

	var web3Service *powchain.Web3Service
	if err := b.services.FetchService(&web3Service); err != nil {
		return err
	}

	var syncService *rbcsync.Service
	if err := b.services.FetchService(&syncService); err != nil {
		return err
	}

	port := ctx.GlobalString(utils.RPCPort.Name)
	cert := ctx.GlobalString(utils.CertFlag.Name)
	key := ctx.GlobalString(utils.KeyFlag.Name)
	rpcService := rpc.NewRPCService(context.Background(), &rpc.Config{
		Port:             port,
		CertFlag:         cert,
		KeyFlag:          key,
		BeaconDB:         b.db,
		Broadcaster:      p2pService,
		ChainService:     chainService,
		OperationService: operationService,
		POWChainService:  web3Service,
		SyncService:      syncService,
	})

	return b.services.RegisterService(rpcService)
}

func (b *BeaconNode) registerPrometheusService(ctx *cli.Context) error {
	service := prometheus.NewPrometheusService(
		fmt.Sprintf(":%d", ctx.GlobalInt64(cmd.MonitoringPortFlag.Name)),
		b.services,
	)
	hook := prometheus.NewLogrusCollector()
	logrus.AddHook(hook)
	return b.services.RegisterService(service)
}

func (b *BeaconNode) registerAttestationService() error {
	attsService := attestation.NewAttestationService(context.Background(),
		&attestation.Config{
			BeaconDB: b.db,
		})

	return b.services.RegisterService(attsService)
}

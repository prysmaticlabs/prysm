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
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/dbcleanup"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/simulator"
	rbcsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "node")
var beaconChainDBName = "beaconchaindata"

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
	registry := shared.NewServiceRegistry()

	beacon := &BeaconNode{
		ctx:      ctx,
		services: registry,
		stop:     make(chan struct{}),
	}

	// Use demo config values if demo config flag is set.
	if ctx.GlobalBool(utils.DemoConfigFlag.Name) {
		params.UseDemoBeaconConfig()
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

	if err := beacon.registerBlockchainService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerDBCleanService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerAttestationService(); err != nil {
		return nil, err
	}

	if err := beacon.registerSimulatorService(ctx); err != nil {
		return nil, err
	}

	if err := beacon.registerSyncService(); err != nil {
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

	log.Info("Starting beacon node")

	b.services.StartAll()

	stop := b.stop
	b.lock.Unlock()

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		go b.Close()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Info("Already shutting down, interrupt more to panic", "times", i-1)
			}
		}
		debug.Exit(b.ctx) // Ensure trace and CPU profile data are flushed.
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
	var genesisJSON string
	if ctx.GlobalIsSet(utils.GenesisJSON.Name) {
		genesisJSON = ctx.GlobalString(utils.GenesisJSON.Name)
	}

	db, err := db.NewDB(path.Join(baseDir, beaconChainDBName))
	if err != nil {
		return err
	}

	log.Info("checking db")

	beaconState, err := db.GetState()
	if err != nil {
		return err
	}
	// Ensure that state has been initialized.
	if beaconState == nil {
		var genesisValidatorRegistry []*pb.ValidatorRecord
		if genesisJSON != "" {
			log.Infof("Initializing Crystallized State from %s", genesisJSON)
			genesisValidatorRegistry, err = utils.InitialValidatorRegistryFromJSON(genesisJSON)
			if err != nil {
				return err
			}
		}
		if err := db.InitializeState(genesisValidatorRegistry); err != nil {
			return err
		}
	}

	beaconState, err = db.GetState()
	if err != nil {
		return err
	}

	hash, err := state.Hash(beaconState)
	if err != nil {
		return err
	}

	genesisBlock, err := db.GetBlockBySlot(0)
	if err != nil {
		return err
	}

	if genesisBlock == nil {
		if err := db.SaveBlock(blocks.NewGenesisBlock(hash[:])); err != nil {
			return err
		}
	}

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

func (b *BeaconNode) registerBlockchainService(ctx *cli.Context) error {
	var web3Service *powchain.Web3Service
	enablePOWChain := ctx.GlobalBool(utils.EnablePOWChain.Name)
	if enablePOWChain {
		if err := b.services.FetchService(&web3Service); err != nil {
			return err
		}
	}

	blockchainService, err := blockchain.NewChainService(context.TODO(), &blockchain.Config{
		BeaconDB:         b.db,
		Web3Service:      web3Service,
		BeaconBlockBuf:   10,
		IncomingBlockBuf: 100, // Big buffer to accommodate other feed subscribers.
		EnablePOWChain:   enablePOWChain,
	})
	if err != nil {
		return fmt.Errorf("could not register blockchain service: %v", err)
	}
	return b.services.RegisterService(blockchainService)
}

func (b *BeaconNode) registerDBCleanService(ctx *cli.Context) error {
	if !ctx.GlobalBool(utils.EnableDBCleanup.Name) {
		return nil
	}

	var chainService *blockchain.ChainService
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	dbCleanService := dbcleanup.NewCleanupService(context.TODO(), &dbcleanup.Config{
		SubscriptionBuf: 100,
		BeaconDB:        b.db,
		ChainService:    chainService,
	})

	return b.services.RegisterService(dbCleanService)
}

func (b *BeaconNode) registerAttestationService() error {
	attestationService := attestation.NewAttestationService(context.TODO(), &attestation.Config{
		BeaconDB: b.db,
	})

	return b.services.RegisterService(attestationService)
}

func (b *BeaconNode) registerPOWChainService(ctx *cli.Context) error {
	if !ctx.GlobalBool(utils.EnablePOWChain.Name) {
		return nil
	}

	rpcClient, err := gethRPC.Dial(b.ctx.GlobalString(utils.Web3ProviderFlag.Name))
	if err != nil {
		log.Fatalf("Access to PoW chain is required for validator. Unable to connect to Geth node: %v", err)
	}
	powClient := ethclient.NewClient(rpcClient)

	web3Service, err := powchain.NewWeb3Service(context.TODO(), &powchain.Web3ServiceConfig{
		Endpoint: b.ctx.GlobalString(utils.Web3ProviderFlag.Name),
		Pubkey:   b.ctx.GlobalString(utils.PubKeyFlag.Name),
		VrcAddr:  common.HexToAddress(b.ctx.GlobalString(utils.VrcContractFlag.Name)),
		Client:   powClient,
		Reader:   powClient,
		Logger:   powClient,
	})
	if err != nil {
		return fmt.Errorf("could not register proof-of-work chain web3Service: %v", err)
	}
	return b.services.RegisterService(web3Service)
}

func (b *BeaconNode) registerSyncService() error {
	var chainService *blockchain.ChainService
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var p2pService *p2p.Server
	if err := b.services.FetchService(&p2pService); err != nil {
		return err
	}

	var attestationService *attestation.Service
	if err := b.services.FetchService(&attestationService); err != nil {
		return err
	}

	cfg := &rbcsync.Config{
		ChainService:  chainService,
		P2P:           p2pService,
		BeaconDB:      b.db,
		AttestService: attestationService,
	}

	syncService := rbcsync.NewSyncService(context.Background(), cfg)
	return b.services.RegisterService(syncService)
}

func (b *BeaconNode) registerSimulatorService(ctx *cli.Context) error {
	if !ctx.GlobalBool(utils.SimulatorFlag.Name) {
		return nil
	}
	var p2pService *p2p.Server
	if err := b.services.FetchService(&p2pService); err != nil {
		return err
	}

	var web3Service *powchain.Web3Service
	var enablePOWChain = ctx.GlobalBool(utils.EnablePOWChain.Name)
	if enablePOWChain {
		if err := b.services.FetchService(&web3Service); err != nil {
			return err
		}
	}

	var chainService *blockchain.ChainService
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	defaultConf := simulator.DefaultConfig()
	cfg := &simulator.Config{
		BlockRequestBuf: defaultConf.BlockRequestBuf,
		BeaconDB:        b.db,
		P2P:             p2pService,
		Web3Service:     web3Service,
		EnablePOWChain:  enablePOWChain,
	}
	simulatorService := simulator.NewSimulator(context.TODO(), cfg)
	return b.services.RegisterService(simulatorService)
}

func (b *BeaconNode) registerRPCService(ctx *cli.Context) error {
	var chainService *blockchain.ChainService
	if err := b.services.FetchService(&chainService); err != nil {
		return err
	}

	var attestationService *attestation.Service
	if err := b.services.FetchService(&attestationService); err != nil {
		return err
	}

	var web3Service *powchain.Web3Service
	var enablePOWChain = ctx.GlobalBool(utils.EnablePOWChain.Name)
	if enablePOWChain {
		if err := b.services.FetchService(&web3Service); err != nil {
			return err
		}
	}

	port := ctx.GlobalString(utils.RPCPort.Name)
	cert := ctx.GlobalString(utils.CertFlag.Name)
	key := ctx.GlobalString(utils.KeyFlag.Name)
	rpcService := rpc.NewRPCService(context.TODO(), &rpc.Config{
		Port:               port,
		CertFlag:           cert,
		KeyFlag:            key,
		SubscriptionBuf:    100,
		BeaconDB:           b.db,
		ChainService:       chainService,
		AttestationService: attestationService,
		POWChainService:    web3Service,
		EnablePOWChain:     enablePOWChain,
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

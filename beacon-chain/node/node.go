// Package node is the main service which launches a beacon node and manages
// the lifecycle of all its associated services at runtime, such as p2p, RPC, sync,
// gracefully closing them if the process ends.
package node

import (
	"bytes"
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/beacon-chain/orchestrator"

	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
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
	regularsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/backuputil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prereq"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

const testSkipPowFlag = "test-skip-pow"

// BeaconNode defines a struct that handles the services running a random beacon chain
// full PoS node. It handles the lifecycle of the entire system and registers
// services to a service registry.
type BeaconNode struct {
	cliCtx          *cli.Context
	ctx             context.Context
	cancel          context.CancelFunc
	services        *shared.ServiceRegistry
	lock            sync.RWMutex
	stop            chan struct{} // Channel to wait for termination notifications.
	db              db.Database
	attestationPool attestations.Pool
	exitPool        voluntaryexits.PoolManager
	slashingsPool   slashings.PoolManager
	depositCache    *depositcache.DepositCache
	stateFeed       *event.Feed
	blockFeed       *event.Feed
	opFeed          *event.Feed
	forkChoiceStore forkchoice.ForkChoicer
	stateGen        *stategen.State
}

// New creates a new node instance, sets up configuration options, and registers
// every required service to the node.
func New(cliCtx *cli.Context) (*BeaconNode, error) {
	if err := tracing.Setup(
		"beacon-chain", // service name
		cliCtx.String(cmd.TracingProcessNameFlag.Name),
		cliCtx.String(cmd.TracingEndpointFlag.Name),
		cliCtx.Float64(cmd.TraceSampleFractionFlag.Name),
		cliCtx.Bool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}

	// Warn if user's platform is not supported
	prereq.WarnIfNotSupported(cliCtx.Context)

	featureconfig.ConfigureBeaconChain(cliCtx)
	cmd.ConfigureBeaconChain(cliCtx)
	flags.ConfigureGlobalFlags(cliCtx)

	if cliCtx.IsSet(cmd.ChainConfigFileFlag.Name) {
		chainConfigFileName := cliCtx.String(cmd.ChainConfigFileFlag.Name)
		params.LoadChainConfigFile(chainConfigFileName)
	}

	if cliCtx.Bool(flags.HistoricalSlasherNode.Name) {
		c := params.BeaconConfig()
		// Save a state every 4 epochs.
		c.SlotsPerArchivedPoint = params.BeaconConfig().SlotsPerEpoch * 4
		params.OverrideBeaconConfig(c)
		cmdConfig := cmd.Get()
		// Allow up to 4096 attestations at a time to be requested from the beacon nde.
		cmdConfig.MaxRPCPageSize = int(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations))
		cmd.Init(cmdConfig)
		log.Warnf(
			"Setting %d slots per archive point and %d max RPC page size for historical slasher usage. This requires additional storage",
			c.SlotsPerArchivedPoint,
			cmdConfig.MaxRPCPageSize,
		)
	}

	if cliCtx.IsSet(flags.SlotsPerArchivedPoint.Name) {
		c := params.BeaconConfig()
		c.SlotsPerArchivedPoint = types.Slot(cliCtx.Int(flags.SlotsPerArchivedPoint.Name))
		params.OverrideBeaconConfig(c)
	}

	// ETH PoW related flags.
	if cliCtx.IsSet(flags.ChainID.Name) {
		c := params.BeaconConfig()
		c.DepositChainID = cliCtx.Uint64(flags.ChainID.Name)
		params.OverrideBeaconConfig(c)
	}
	if cliCtx.IsSet(flags.NetworkID.Name) {
		c := params.BeaconConfig()
		c.DepositNetworkID = cliCtx.Uint64(flags.NetworkID.Name)
		params.OverrideBeaconConfig(c)
	}
	if cliCtx.IsSet(flags.DepositContractFlag.Name) {
		c := params.BeaconConfig()
		c.DepositContractAddress = cliCtx.String(flags.DepositContractFlag.Name)
		params.OverrideBeaconConfig(c)
	}

	// Setting chain network specific flags.
	if cliCtx.IsSet(cmd.BootstrapNode.Name) {
		c := params.BeaconNetworkConfig()
		c.BootstrapNodes = cliCtx.StringSlice(cmd.BootstrapNode.Name)
		params.OverrideBeaconNetworkConfig(c)
	}
	if cliCtx.IsSet(flags.ContractDeploymentBlock.Name) {
		networkCfg := params.BeaconNetworkConfig()
		networkCfg.ContractDeploymentBlock = uint64(cliCtx.Int(flags.ContractDeploymentBlock.Name))
		params.OverrideBeaconNetworkConfig(networkCfg)
	}

	registry := shared.NewServiceRegistry()

	ctx, cancel := context.WithCancel(cliCtx.Context)
	beacon := &BeaconNode{
		cliCtx:          cliCtx,
		ctx:             ctx,
		cancel:          cancel,
		services:        registry,
		stop:            make(chan struct{}),
		stateFeed:       new(event.Feed),
		blockFeed:       new(event.Feed),
		opFeed:          new(event.Feed),
		attestationPool: attestations.NewPool(),
		exitPool:        voluntaryexits.NewPool(),
		slashingsPool:   slashings.NewPool(),
	}

	if err := beacon.startDB(cliCtx); err != nil {
		return nil, err
	}

	beacon.startStateGen()

	if err := beacon.registerP2P(cliCtx); err != nil {
		return nil, err
	}

	if err := beacon.registerPOWChainService(); err != nil {
		return nil, err
	}

	if err := beacon.registerAttestationPool(); err != nil {
		return nil, err
	}

	if err := beacon.registerInteropServices(); err != nil {
		return nil, err
	}

	beacon.startForkChoice()

	if err := beacon.registerBlockchainService(); err != nil {
		return nil, err
	}

	if err := beacon.registerInitialSyncService(); err != nil {
		return nil, err
	}

	if err := beacon.registerSyncService(); err != nil {
		return nil, err
	}

	if err := beacon.registerRPCService(); err != nil {
		return nil, err
	}

	if err := beacon.registerGRPCGateway(); err != nil {
		return nil, err
	}

	if !cliCtx.Bool(cmd.DisableMonitoringFlag.Name) {
		if err := beacon.registerPrometheusService(cliCtx); err != nil {
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
	b.cancel()
	close(b.stop)
}

func (b *BeaconNode) startForkChoice() {
	f := protoarray.New(0, 0, params.BeaconConfig().ZeroHash)
	b.forkChoiceStore = f
}

func (b *BeaconNode) startDB(cliCtx *cli.Context) error {
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
	return nil
}

func (b *BeaconNode) startStateGen() {
	b.stateGen = stategen.New(b.db)
}

func readbootNodes(fileName string) ([]string, error) {
	fileContent, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	listNodes := make([]string, 0)
	err = yaml.Unmarshal(fileContent, &listNodes)
	if err != nil {
		return nil, err
	}
	return listNodes, nil
}

func (b *BeaconNode) registerP2P(cliCtx *cli.Context) error {
	// Bootnode ENR may be a filepath to a YAML file
	bootnodesTemp := params.BeaconNetworkConfig().BootstrapNodes // actual CLI values
	bootnodeAddrs := make([]string, 0)                           // dest of final list of nodes
	for _, addr := range bootnodesTemp {
		if filepath.Ext(addr) == ".yaml" {
			fileNodes, err := readbootNodes(addr)
			if err != nil {
				return err
			}
			bootnodeAddrs = append(bootnodeAddrs, fileNodes...)
		} else {
			bootnodeAddrs = append(bootnodeAddrs, addr)
		}
	}

	datadir := cliCtx.String(cmd.DataDirFlag.Name)
	if datadir == "" {
		datadir = cmd.DefaultDataDir()
		if datadir == "" {
			log.Fatal(
				"Could not determine your system's HOME path, please specify a --datadir you wish " +
					"to use for your chain data",
			)
		}
	}

	svc, err := p2p.NewService(b.ctx, &p2p.Config{
		NoDiscovery:       cliCtx.Bool(cmd.NoDiscovery.Name),
		StaticPeers:       sliceutil.SplitCommaSeparated(cliCtx.StringSlice(cmd.StaticPeers.Name)),
		BootstrapNodeAddr: bootnodeAddrs,
		RelayNodeAddr:     cliCtx.String(cmd.RelayNode.Name),
		DataDir:           datadir,
		LocalIP:           cliCtx.String(cmd.P2PIP.Name),
		HostAddress:       cliCtx.String(cmd.P2PHost.Name),
		HostDNS:           cliCtx.String(cmd.P2PHostDNS.Name),
		PrivateKey:        cliCtx.String(cmd.P2PPrivKey.Name),
		MetaDataDir:       cliCtx.String(cmd.P2PMetadata.Name),
		TCPPort:           cliCtx.Uint(cmd.P2PTCPPort.Name),
		UDPPort:           cliCtx.Uint(cmd.P2PUDPPort.Name),
		MaxPeers:          cliCtx.Uint(cmd.P2PMaxPeers.Name),
		AllowListCIDR:     cliCtx.String(cmd.P2PAllowList.Name),
		DenyListCIDR:      sliceutil.SplitCommaSeparated(cliCtx.StringSlice(cmd.P2PDenyList.Name)),
		EnableUPnP:        cliCtx.Bool(cmd.EnableUPnPFlag.Name),
		DisableDiscv5:     cliCtx.Bool(flags.DisableDiscv5.Name),
		StateNotifier:     b,
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

	var opsService *attestations.Service
	if err := b.services.FetchService(&opsService); err != nil {
		return err
	}

	wsp := b.cliCtx.String(flags.WeakSubjectivityCheckpt.Name)
	bRoot, epoch, err := convertWspInput(wsp)
	if err != nil {
		return err
	}

	var orcClient *orchestrator.RPCClient
	if b.cliCtx.Bool(cmd.VanguardNetwork.Name) {
		endpoint := b.cliCtx.String(flags.OrcRpcProviderFlag.Name)
		if endpoint == "" {
			log.Error("No orchestrator node specified to run with the vanguard node. Please consider running your own orchestrator node for final consensus.")
		}

		orcClient, err = orchestrator.Dial(endpoint)
		if err != nil {
			log.WithError(err).Error("Failed to create orchestrator rpc client")
			return err
		}
	}

	maxRoutines := b.cliCtx.Int(cmd.MaxGoroutines.Name)
	blockchainService, err := blockchain.NewService(b.ctx, &blockchain.Config{
		BeaconDB:          b.db,
		DepositCache:      b.depositCache,
		ChainStartFetcher: web3Service,
		AttPool:           b.attestationPool,
		ExitPool:          b.exitPool,
		SlashingPool:      b.slashingsPool,
		P2p:               b.fetchP2P(),
		MaxRoutines:       maxRoutines,
		StateNotifier:     b,
		BlockNotifier:     b,
		ForkChoiceStore:   b.forkChoiceStore,
		OpsService:        opsService,
		StateGen:          b.stateGen,
		WspBlockRoot:      bRoot,
		WspEpoch:          epoch,

		// vanguard: EnableVanguardNode and OrcRPCClient is used for vanguard chain
		EnableVanguardNode: b.cliCtx.Bool(cmd.VanguardNetwork.Name),
		OrcRPCClient:       orcClient,
	})
	if err != nil {
		return errors.Wrap(err, "could not register blockchain service")
	}
	return b.services.RegisterService(blockchainService)
}

func (b *BeaconNode) registerPOWChainService() error {
	if b.cliCtx.Bool(testSkipPowFlag) {
		return b.services.RegisterService(&powchain.Service{})
	}
	depAddress := params.BeaconConfig().DepositContractAddress
	if depAddress == "" {
		log.Fatal("Valid deposit contract is required")
	}

	if !common.IsHexAddress(depAddress) {
		log.Fatalf("Invalid deposit contract address given: %s", depAddress)
	}

	if b.cliCtx.String(flags.HTTPWeb3ProviderFlag.Name) == "" {
		log.Error("No ETH1 node specified to run with the beacon node. Please consider running your own ETH1 node for better uptime, security, and decentralization of ETH2. Visit https://docs.prylabs.network/docs/prysm-usage/setup-eth1 for more information.")
		log.Error("You will need to specify --http-web3provider to attach an eth1 node to the prysm node. Without an eth1 node block proposals for your validator will be affected and the beacon node will not be able to initialize the genesis state.")
	}
	endpoints := []string{b.cliCtx.String(flags.HTTPWeb3ProviderFlag.Name)}
	endpoints = append(endpoints, b.cliCtx.StringSlice(flags.FallbackWeb3ProviderFlag.Name)...)

	cfg := &powchain.Web3ServiceConfig{
		HTTPEndpoints:      endpoints,
		DepositContract:    common.HexToAddress(depAddress),
		BeaconDB:           b.db,
		DepositCache:       b.depositCache,
		StateNotifier:      b,
		StateGen:           b.stateGen,
		Eth1HeaderReqLimit: b.cliCtx.Uint64(flags.Eth1HeaderReqLimit.Name),
	}
	web3Service, err := powchain.NewService(b.ctx, cfg)
	if err != nil {
		return errors.Wrap(err, "could not register proof-of-work chain web3Service")
	}
	knownContract, err := b.db.DepositContractAddress(b.ctx)
	if err != nil {
		return err
	}
	if len(knownContract) == 0 {
		if err := b.db.SaveDepositContractAddress(b.ctx, cfg.DepositContract); err != nil {
			return errors.Wrap(err, "could not save deposit contract")
		}
	}
	if len(knownContract) > 0 && !bytes.Equal(cfg.DepositContract.Bytes(), knownContract) {
		return fmt.Errorf("database contract is %#x but tried to run with %#x. This likely means "+
			"you are trying to run on a different network than what the database contains. You can run once with "+
			"'--clear-db' to wipe the old database or use an alternative data directory with '--datadir'",
			knownContract, cfg.DepositContract.Bytes())
	}

	log.Infof("Deposit contract: %#x", cfg.DepositContract.Bytes())
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

	rs := regularsync.NewService(b.ctx, &regularsync.Config{
		DB:                  b.db,
		P2P:                 b.fetchP2P(),
		Chain:               chainService,
		InitialSync:         initSync,
		StateNotifier:       b,
		BlockNotifier:       b,
		AttestationNotifier: b,
		AttPool:             b.attestationPool,
		ExitPool:            b.exitPool,
		SlashingPool:        b.slashingsPool,
		StateGen:            b.stateGen,
	})

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
	enableDebugRPCEndpoints := b.cliCtx.Bool(flags.EnableDebugRPCEndpoints.Name)
	maxMsgSize := b.cliCtx.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name)
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
		POWChainService:         web3Service,
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

		// vanguard: EnableVanguardNode and UnconfirmedBlockFetcher is used for vanguard chain
		EnableVanguardNode:      b.cliCtx.Bool(cmd.VanguardNetwork.Name),
		UnconfirmedBlockFetcher: chainService,
		PendingQueueFetcher:     chainService,
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
				Handler: backuputil.BackupHandler(b.db, cliCtx.String(cmd.BackupWebhookOutputDir.Name)),
			},
		)
	}

	additionalHandlers = append(additionalHandlers, prometheus.Handler{Path: "/tree", Handler: c.TreeHandler})

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
	return b.services.RegisterService(
		gateway.New(
			b.ctx,
			selfAddress,
			selfCert,
			gatewayAddress,
			nil, /*optional mux*/
			allowedOrigins,
			enableDebugRPCEndpoints,
			b.cliCtx.Uint64(cmd.GrpcMaxCallRecvMsgSizeFlag.Name),
		),
	)
}

func (b *BeaconNode) registerInteropServices() error {
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

		return b.services.RegisterService(svc)
	}
	return nil
}

// Package node is the main process which handles the lifecycle of
// the runtime services in a validator client process, gracefully shutting
// everything down upon close.
package node

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prereq"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/client"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/prysmaticlabs/prysm/validator/rpc"
	"github.com/prysmaticlabs/prysm/validator/rpc/gateway"
	slashing_protection "github.com/prysmaticlabs/prysm/validator/slashing-protection"
	"github.com/prysmaticlabs/prysm/validator/web"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "node")

// ValidatorClient defines an instance of an eth2 validator that manages
// the entire lifecycle of services attached to it participating in eth2.
type ValidatorClient struct {
	cliCtx            *cli.Context
	db                *kv.Store
	services          *shared.ServiceRegistry // Lifecycle and service store.
	lock              sync.RWMutex
	wallet            *wallet.Wallet
	walletInitialized *event.Feed
	stop              chan struct{} // Channel to wait for termination notifications.
}

// NewValidatorClient creates a new, Prysm validator client.
func NewValidatorClient(cliCtx *cli.Context) (*ValidatorClient, error) {
	if err := tracing.Setup(
		"validator", // service name
		cliCtx.String(cmd.TracingProcessNameFlag.Name),
		cliCtx.String(cmd.TracingEndpointFlag.Name),
		cliCtx.Float64(cmd.TraceSampleFractionFlag.Name),
		cliCtx.Bool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}

	verbosity := cliCtx.String(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return nil, err
	}
	logrus.SetLevel(level)

	// Warn if user's platform is not supported
	prereq.WarnIfNotSupported(cliCtx.Context)

	registry := shared.NewServiceRegistry()
	ValidatorClient := &ValidatorClient{
		cliCtx:            cliCtx,
		services:          registry,
		walletInitialized: new(event.Feed),
		stop:              make(chan struct{}),
	}

	featureconfig.ConfigureValidator(cliCtx)
	cmd.ConfigureValidator(cliCtx)

	if cliCtx.IsSet(cmd.ChainConfigFileFlag.Name) {
		chainConfigFileName := cliCtx.String(cmd.ChainConfigFileFlag.Name)
		params.LoadChainConfigFile(chainConfigFileName)
	}

	// If the --web flag is enabled to administer the validator
	// client via a web portal, we start the validator client in a different way.
	if cliCtx.IsSet(flags.EnableWebFlag.Name) {
		log.Info("Enabling web portal to manage the validator client")
		if err := ValidatorClient.initializeForWeb(cliCtx); err != nil {
			return nil, err
		}
		return ValidatorClient, nil
	}
	if err := ValidatorClient.initializeFromCLI(cliCtx); err != nil {
		return nil, err
	}
	if err := ValidatorClient.db.MigrateV2ProposalsProtectionDb(cliCtx.Context); err != nil {
		return nil, err
	}
	return ValidatorClient, nil
}

// Start every service in the validator client.
func (s *ValidatorClient) Start() {
	s.lock.Lock()

	log.WithFields(logrus.Fields{
		"version": version.GetVersion(),
	}).Info("Starting validator node")

	s.services.StartAll()

	stop := s.stop
	s.lock.Unlock()

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		debug.Exit(s.cliCtx) // Ensure trace and CPU profile data are flushed.
		go s.Close()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.WithField("times", i-1).Info("Already shutting down, interrupt more to panic.")
			}
		}
		panic("Panic closing the validator client")
	}()

	// Wait for stop channel to be closed.
	<-stop
}

// Close handles graceful shutdown of the system.
func (s *ValidatorClient) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.services.StopAll()
	log.Info("Stopping Prysm validator")
	if !s.cliCtx.IsSet(flags.InteropNumValidators.Name) {
		if err := s.wallet.UnlockWalletConfigFile(); err != nil {
			log.WithError(err).Errorf("Failed to unlock wallet config file.")
		}
	}
	close(s.stop)
}

func (s *ValidatorClient) initializeFromCLI(cliCtx *cli.Context) error {
	var keyManager keymanager.IKeymanager
	var err error
	var accountsDir string
	if cliCtx.IsSet(flags.InteropNumValidators.Name) {
		numValidatorKeys := cliCtx.Uint64(flags.InteropNumValidators.Name)
		offset := cliCtx.Uint64(flags.InteropStartIndex.Name)
		keyManager, err = imported.NewInteropKeymanager(cliCtx.Context, offset, numValidatorKeys)
		if err != nil {
			return errors.Wrap(err, "could not generate interop keys")
		}
	} else {
		// Read the wallet from the specified path.
		w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
			return nil, errors.New("no wallet found, create a new one with validator wallet create")
		})
		if err != nil {
			return errors.Wrap(err, "could not open wallet")
		}
		s.wallet = w
		log.WithFields(logrus.Fields{
			"wallet":          w.AccountsDir(),
			"keymanager-kind": w.KeymanagerKind().String(),
		}).Info("Opened validator wallet")
		keyManager, err = w.InitializeKeymanager(
			cliCtx.Context, false, /* skipMnemonicConfirm */
		)
		if err != nil {
			return errors.Wrap(err, "could not read keymanager for wallet")
		}
		if err := w.LockWalletConfigFile(cliCtx.Context); err != nil {
			log.Fatalf("Could not get a lock on wallet file. Please check if you have another validator instance running and using the same wallet: %v", err)
		}
		accountsDir = s.wallet.AccountsDir()
	}

	dataDir := moveDb(cliCtx, accountsDir)
	clearFlag := cliCtx.Bool(cmd.ClearDB.Name)
	forceClearFlag := cliCtx.Bool(cmd.ForceClearDB.Name)
	if clearFlag || forceClearFlag {
		if dataDir == "" {
			dataDir = cmd.DefaultDataDir()
			if dataDir == "" {
				log.Fatal(
					"Could not determine your system's HOME path, please specify a --datadir you wish " +
						"to use for your validator data",
				)
			}

		}
		if err := clearDB(dataDir, forceClearFlag); err != nil {
			return err
		}
	}
	log.WithField("databasePath", dataDir).Info("Checking DB")

	valDB, err := kv.NewKVStore(dataDir, nil)
	if err != nil {
		return errors.Wrap(err, "could not initialize db")
	}
	s.db = valDB
	if !cliCtx.Bool(cmd.DisableMonitoringFlag.Name) {
		if err := s.registerPrometheusService(); err != nil {
			return err
		}
	}
	if featureconfig.Get().SlasherProtection {
		if err := s.registerSlasherClientService(); err != nil {
			return err
		}
	}
	if err := s.registerClientService(keyManager); err != nil {
		return err
	}
	if cliCtx.Bool(flags.EnableRPCFlag.Name) {
		if err := s.registerRPCService(cliCtx); err != nil {
			return err
		}
		if err := s.registerRPCGatewayService(cliCtx); err != nil {
			return err
		}
	}
	return nil
}

func moveDb(cliCtx *cli.Context, accountsDir string) string {
	dataDir := cliCtx.String(cmd.DataDirFlag.Name)
	if accountsDir != "" {
		dataFile := filepath.Join(dataDir, kv.ProtectionDbFileName)
		newDataFile := filepath.Join(accountsDir, kv.ProtectionDbFileName)
		if fileutil.FileExists(dataFile) && !fileutil.FileExists(newDataFile) {
			log.WithFields(logrus.Fields{
				"oldDbPath": dataDir,
				"walletDir": accountsDir,
			}).Info("Moving validator protection db to wallet dir")
			err := fileutil.CopyFile(dataFile, newDataFile)
			if err != nil {
				log.Fatal(err)
			}
		}
		dataDir = accountsDir
	}
	return dataDir
}

func (s *ValidatorClient) initializeForWeb(cliCtx *cli.Context) error {
	clearFlag := cliCtx.Bool(cmd.ClearDB.Name)
	forceClearFlag := cliCtx.Bool(cmd.ForceClearDB.Name)
	dataDir := cliCtx.String(cmd.DataDirFlag.Name)
	if clearFlag || forceClearFlag {
		if dataDir == "" {
			dataDir = cmd.DefaultDataDir()
			if dataDir == "" {
				log.Fatal(
					"Could not determine your system's HOME path, please specify a --datadir you wish " +
						"to use for your validator data",
				)
			}

		}
		if err := clearDB(dataDir, forceClearFlag); err != nil {
			return err
		}
	}
	log.WithField("databasePath", dataDir).Info("Checking DB")
	valDB, err := kv.NewKVStore(dataDir, make([][48]byte, 0))
	if err != nil {
		return errors.Wrap(err, "could not initialize db")
	}
	s.db = valDB
	if !cliCtx.Bool(cmd.DisableMonitoringFlag.Name) {
		if err := s.registerPrometheusService(); err != nil {
			return err
		}
	}
	if featureconfig.Get().SlasherProtection {
		if err := s.registerSlasherClientService(); err != nil {
			return err
		}
	}
	if err := s.registerClientService(nil); err != nil {
		return err
	}
	if err := s.registerRPCService(cliCtx); err != nil {
		return err
	}
	if err := s.registerRPCGatewayService(cliCtx); err != nil {
		return err
	}
	return s.registerWebService(cliCtx)
}

func (s *ValidatorClient) registerPrometheusService() error {
	service := prometheus.NewService(
		fmt.Sprintf("%s:%d", s.cliCtx.String(cmd.MonitoringHostFlag.Name), s.cliCtx.Int(flags.MonitoringPortFlag.Name)),
		s.services,
	)
	logrus.AddHook(prometheus.NewLogrusCollector())
	return s.services.RegisterService(service)
}

func (s *ValidatorClient) registerClientService(
	keyManager keymanager.IKeymanager,
) error {
	endpoint := s.cliCtx.String(flags.BeaconRPCProviderFlag.Name)
	dataDir := s.cliCtx.String(cmd.DataDirFlag.Name)
	logValidatorBalances := !s.cliCtx.Bool(flags.DisablePenaltyRewardLogFlag.Name)
	emitAccountMetrics := !s.cliCtx.Bool(flags.DisableAccountMetricsFlag.Name)
	cert := s.cliCtx.String(flags.CertFlag.Name)
	graffiti := s.cliCtx.String(flags.GraffitiFlag.Name)
	maxCallRecvMsgSize := s.cliCtx.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name)
	grpcRetries := s.cliCtx.Uint(flags.GrpcRetriesFlag.Name)
	grpcRetryDelay := s.cliCtx.Duration(flags.GrpcRetryDelayFlag.Name)
	var sp *slashing_protection.Service
	var protector slashing_protection.Protector
	if err := s.services.FetchService(&sp); err == nil {
		protector = sp
	}
	v, err := client.NewValidatorService(s.cliCtx.Context, &client.Config{
		Endpoint:                   endpoint,
		DataDir:                    dataDir,
		KeyManager:                 keyManager,
		LogValidatorBalances:       logValidatorBalances,
		EmitAccountMetrics:         emitAccountMetrics,
		CertFlag:                   cert,
		GraffitiFlag:               graffiti,
		GrpcMaxCallRecvMsgSizeFlag: maxCallRecvMsgSize,
		GrpcRetriesFlag:            grpcRetries,
		GrpcRetryDelay:             grpcRetryDelay,
		GrpcHeadersFlag:            s.cliCtx.String(flags.GrpcHeadersFlag.Name),
		Protector:                  protector,
		ValDB:                      s.db,
		UseWeb:                     s.cliCtx.Bool(flags.EnableWebFlag.Name),
		WalletInitializedFeed:      s.walletInitialized,
	})

	if err != nil {
		return errors.Wrap(err, "could not initialize client service")
	}
	return s.services.RegisterService(v)
}
func (s *ValidatorClient) registerSlasherClientService() error {
	endpoint := s.cliCtx.String(flags.SlasherRPCProviderFlag.Name)
	if endpoint == "" {
		return errors.New("external slasher feature flag is set but no slasher endpoint is configured")

	}
	cert := s.cliCtx.String(flags.SlasherCertFlag.Name)
	maxCallRecvMsgSize := s.cliCtx.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name)
	grpcRetries := s.cliCtx.Uint(flags.GrpcRetriesFlag.Name)
	grpcRetryDelay := s.cliCtx.Duration(flags.GrpcRetryDelayFlag.Name)
	sp, err := slashing_protection.NewService(s.cliCtx.Context, &slashing_protection.Config{
		Endpoint:                   endpoint,
		CertFlag:                   cert,
		GrpcMaxCallRecvMsgSizeFlag: maxCallRecvMsgSize,
		GrpcRetriesFlag:            grpcRetries,
		GrpcRetryDelay:             grpcRetryDelay,
		GrpcHeadersFlag:            s.cliCtx.String(flags.GrpcHeadersFlag.Name),
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize client service")
	}
	return s.services.RegisterService(sp)
}

func (s *ValidatorClient) registerRPCService(cliCtx *cli.Context) error {
	var vs *client.ValidatorService
	if err := s.services.FetchService(&vs); err != nil {
		return err
	}
	rpcHost := cliCtx.String(flags.RPCHost.Name)
	rpcPort := cliCtx.Int(flags.RPCPort.Name)
	nodeGatewayEndpoint := cliCtx.String(flags.BeaconRPCGatewayProviderFlag.Name)
	walletDir := cliCtx.String(flags.WalletDirFlag.Name)
	server := rpc.NewServer(cliCtx.Context, &rpc.Config{
		ValDB:                 s.db,
		Host:                  rpcHost,
		Port:                  fmt.Sprintf("%d", rpcPort),
		WalletInitializedFeed: s.walletInitialized,
		ValidatorService:      vs,
		SyncChecker:           vs,
		GenesisFetcher:        vs,
		NodeGatewayEndpoint:   nodeGatewayEndpoint,
		WalletDir:             walletDir,
	})
	return s.services.RegisterService(server)
}

func (s *ValidatorClient) registerRPCGatewayService(cliCtx *cli.Context) error {
	gatewayHost := cliCtx.String(flags.GRPCGatewayHost.Name)
	gatewayPort := cliCtx.Int(flags.GRPCGatewayPort.Name)
	rpcHost := cliCtx.String(flags.RPCHost.Name)
	rpcPort := cliCtx.Int(flags.RPCPort.Name)
	rpcAddr := fmt.Sprintf("%s:%d", rpcHost, rpcPort)
	gatewayAddress := fmt.Sprintf("%s:%d", gatewayHost, gatewayPort)
	allowedOrigins := strings.Split(cliCtx.String(flags.GPRCGatewayCorsDomain.Name), ",")
	gatewaySrv := gateway.New(
		cliCtx.Context,
		rpcAddr,
		gatewayAddress,
		allowedOrigins,
	)
	return s.services.RegisterService(gatewaySrv)
}

func (s *ValidatorClient) registerWebService(cliCtx *cli.Context) error {
	host := cliCtx.String(flags.WebHostFlag.Name)
	port := cliCtx.Uint64(flags.WebPortFlag.Name)
	webAddress := fmt.Sprintf("%s:%d", host, port)
	srv := web.NewServer(webAddress)
	return s.services.RegisterService(srv)
}

func clearDB(dataDir string, force bool) error {
	var err error
	clearDBConfirmed := force

	if !force {
		actionText := "This will delete your validator's historical actions database stored in your data directory. " +
			"This may lead to potential slashing - do you want to proceed? (Y/N)"
		deniedText := "The historical actions database will not be deleted. No changes have been made."
		clearDBConfirmed, err = cmd.ConfirmAction(actionText, deniedText)
		if err != nil {
			return errors.Wrapf(err, "Could not clear DB in dir %s", dataDir)
		}
	}

	if clearDBConfirmed {
		valDB, err := kv.NewKVStore(dataDir, nil)
		if err != nil {
			return errors.Wrapf(err, "Could not create DB in dir %s", dataDir)
		}
		if err := valDB.Close(); err != nil {
			return errors.Wrapf(err, "could not close DB in dir %s", dataDir)
		}

		log.Warning("Removing database")
		if err := valDB.ClearDB(); err != nil {
			return errors.Wrapf(err, "Could not clear DB in dir %s", dataDir)
		}
	}

	return nil
}

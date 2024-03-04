// Package node is the main process which handles the lifecycle of
// the runtime services in a validator client process, gracefully shutting
// everything down upon close.
package node

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	fastssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v5/api"
	"github.com/prysmaticlabs/prysm/v5/api/gateway"
	"github.com/prysmaticlabs/prysm/v5/api/server"
	"github.com/prysmaticlabs/prysm/v5/async/event"
	"github.com/prysmaticlabs/prysm/v5/cmd"
	"github.com/prysmaticlabs/prysm/v5/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/config/proposer"
	"github.com/prysmaticlabs/prysm/v5/config/proposer/loader"
	"github.com/prysmaticlabs/prysm/v5/container/slice"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/io/file"
	"github.com/prysmaticlabs/prysm/v5/monitoring/backup"
	"github.com/prysmaticlabs/prysm/v5/monitoring/prometheus"
	tracing2 "github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	pb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime"
	"github.com/prysmaticlabs/prysm/v5/runtime/debug"
	"github.com/prysmaticlabs/prysm/v5/runtime/prereqs"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v5/validator/client"
	"github.com/prysmaticlabs/prysm/v5/validator/db/iface"
	"github.com/prysmaticlabs/prysm/v5/validator/db/kv"
	g "github.com/prysmaticlabs/prysm/v5/validator/graffiti"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager/local"
	remoteweb3signer "github.com/prysmaticlabs/prysm/v5/validator/keymanager/remote-web3signer"
	"github.com/prysmaticlabs/prysm/v5/validator/rpc"
	"github.com/prysmaticlabs/prysm/v5/validator/web"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/encoding/protojson"
)

// ValidatorClient defines an instance of an Ethereum validator that manages
// the entire lifecycle of services attached to it participating in proof of stake.
type ValidatorClient struct {
	cliCtx            *cli.Context
	ctx               context.Context
	cancel            context.CancelFunc
	db                *kv.Store
	services          *runtime.ServiceRegistry // Lifecycle and service store.
	lock              sync.RWMutex
	wallet            *wallet.Wallet
	walletInitialized *event.Feed
	stop              chan struct{} // Channel to wait for termination notifications.
}

// NewValidatorClient creates a new instance of the Prysm validator client.
func NewValidatorClient(cliCtx *cli.Context) (*ValidatorClient, error) {
	// TODO(#9883) - Maybe we can pass in a new validator client config instead of the cliCTX to abstract away the use of flags here .
	if err := tracing2.Setup(
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
	prereqs.WarnIfPlatformNotSupported(cliCtx.Context)

	registry := runtime.NewServiceRegistry()
	ctx, cancel := context.WithCancel(cliCtx.Context)
	validatorClient := &ValidatorClient{
		cliCtx:            cliCtx,
		ctx:               ctx,
		cancel:            cancel,
		services:          registry,
		walletInitialized: new(event.Feed),
		stop:              make(chan struct{}),
	}

	if err := features.ConfigureValidator(cliCtx); err != nil {
		return nil, err
	}
	if err := cmd.ConfigureValidator(cliCtx); err != nil {
		return nil, err
	}

	if cliCtx.IsSet(cmd.ChainConfigFileFlag.Name) {
		chainConfigFileName := cliCtx.String(cmd.ChainConfigFileFlag.Name)
		if err := params.LoadChainConfigFile(chainConfigFileName, nil); err != nil {
			return nil, err
		}
	}

	configureFastSSZHashingAlgorithm()

	// initialize router used for endpoints
	router := newRouter(cliCtx)
	// If the --web flag is enabled to administer the validator
	// client via a web portal, we start the validator client in a different way.
	// Change Web flag name to enable keymanager API, look at merging initializeFromCLI and initializeForWeb maybe after WebUI DEPRECATED.
	if cliCtx.IsSet(flags.EnableWebFlag.Name) {
		if cliCtx.IsSet(flags.Web3SignerURLFlag.Name) || cliCtx.IsSet(flags.Web3SignerPublicValidatorKeysFlag.Name) {
			log.Warn("Remote Keymanager API enabled. Prysm web does not properly support web3signer at this time")
		}
		log.Info("Enabling web portal to manage the validator client")
		if err := validatorClient.initializeForWeb(cliCtx, router); err != nil {
			return nil, err
		}
		return validatorClient, nil
	}

	if err := validatorClient.initializeFromCLI(cliCtx, router); err != nil {
		return nil, err
	}

	return validatorClient, nil
}

func newRouter(cliCtx *cli.Context) *mux.Router {
	var allowedOrigins []string
	if cliCtx.IsSet(flags.GPRCGatewayCorsDomain.Name) {
		allowedOrigins = strings.Split(cliCtx.String(flags.GPRCGatewayCorsDomain.Name), ",")
	} else {
		allowedOrigins = strings.Split(flags.GPRCGatewayCorsDomain.Value, ",")
	}
	r := mux.NewRouter()
	r.Use(server.NormalizeQueryValuesHandler)
	r.Use(server.CorsHandler(allowedOrigins))
	return r
}

// Start every service in the validator client.
func (c *ValidatorClient) Start() {
	c.lock.Lock()

	log.WithFields(logrus.Fields{
		"version": version.Version(),
	}).Info("Starting validator node")

	c.services.StartAll()

	stop := c.stop
	c.lock.Unlock()

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		debug.Exit(c.cliCtx) // Ensure trace and CPU profile data are flushed.
		go c.Close()
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
func (c *ValidatorClient) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.services.StopAll()
	log.Info("Stopping Prysm validator")
	c.cancel()
	close(c.stop)
}

// checkLegacyDatabaseLocation checks is a database exists in the specified location.
// If it does not, it checks if a database exists in the legacy location.
// If it does, it returns the legacy location.
func (c *ValidatorClient) getLegacyDatabaseLocation(
	isInteropNumValidatorsSet bool,
	isWeb3SignerURLFlagSet bool,
	dataDir string,
	dataFile string,
	walletDir string,
) (string, string) {
	if isInteropNumValidatorsSet || dataDir != cmd.DefaultDataDir() || file.Exists(dataFile) || c.wallet == nil {
		return dataDir, dataFile
	}

	// We look in the previous, legacy directories.
	// See https://github.com/prysmaticlabs/prysm/issues/13391
	legacyDataDir := c.wallet.AccountsDir()
	if isWeb3SignerURLFlagSet {
		legacyDataDir = walletDir
	}

	legacyDataFile := filepath.Join(legacyDataDir, kv.ProtectionDbFileName)

	if file.Exists(legacyDataFile) {
		log.Infof(`Database not found in the --datadir directory (%s)
		but found in the --wallet-dir directory (%s),
		which was the legacy default.
		The next time you run the validator client without a database,
		it will be created into the --datadir directory (%s).
		To silence this message, you can move the database from (%s)
		to (%s).`,
			dataDir, legacyDataDir, dataDir, legacyDataFile, dataFile)

		dataDir = legacyDataDir
		dataFile = legacyDataFile
	}

	return dataDir, dataFile
}

func (c *ValidatorClient) initializeFromCLI(cliCtx *cli.Context, router *mux.Router) error {
	dataDir := cliCtx.String(cmd.DataDirFlag.Name)
	dataFile := filepath.Join(dataDir, kv.ProtectionDbFileName)
	walletDir := cliCtx.String(flags.WalletDirFlag.Name)
	isInteropNumValidatorsSet := cliCtx.IsSet(flags.InteropNumValidators.Name)
	isWeb3SignerURLFlagSet := cliCtx.IsSet(flags.Web3SignerURLFlag.Name)

	if !isInteropNumValidatorsSet {
		// Custom Check For Web3Signer
		if isWeb3SignerURLFlagSet {
			c.wallet = wallet.NewWalletForWeb3Signer()
		} else {
			w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
				return nil, wallet.ErrNoWalletFound
			})
			if err != nil {
				return errors.Wrap(err, "could not open wallet")
			}
			c.wallet = w
			// TODO(#9883) - Remove this when we have a better way to handle this.
			log.WithFields(logrus.Fields{
				"wallet":         w.AccountsDir(),
				"keymanagerKind": w.KeymanagerKind().String(),
			}).Info("Opened validator wallet")
		}
	}

	// Workaround for https://github.com/prysmaticlabs/prysm/issues/13391
	dataDir, dataFile = c.getLegacyDatabaseLocation(
		isInteropNumValidatorsSet,
		isWeb3SignerURLFlagSet,
		dataDir,
		dataFile,
		walletDir,
	)

	clearFlag := cliCtx.Bool(cmd.ClearDB.Name)
	forceClearFlag := cliCtx.Bool(cmd.ForceClearDB.Name)
	if clearFlag || forceClearFlag {
		if err := clearDB(cliCtx.Context, dataDir, forceClearFlag); err != nil {
			return err
		}
	} else {
		if !file.Exists(dataFile) {
			log.Warnf("Slashing protection file %s is missing.\n"+
				"If you changed your --datadir, please copy your previous \"validator.db\" file into your current --datadir.\n"+
				"Disregard this warning if this is the first time you are running this set of keys.", dataFile)
		}
	}
	log.WithField("databasePath", dataDir).Info("Checking DB")

	valDB, err := kv.NewKVStore(cliCtx.Context, dataDir, &kv.Config{
		PubKeys: nil,
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize db")
	}
	c.db = valDB
	if err := valDB.RunUpMigrations(cliCtx.Context); err != nil {
		return errors.Wrap(err, "could not run database migration")
	}

	if !cliCtx.Bool(cmd.DisableMonitoringFlag.Name) {
		if err := c.registerPrometheusService(cliCtx); err != nil {
			return err
		}
	}
	if err := c.registerValidatorService(cliCtx); err != nil {
		return err
	}
	if cliCtx.Bool(flags.EnableRPCFlag.Name) {
		if err := c.registerRPCService(router); err != nil {
			return err
		}
		if err := c.registerRPCGatewayService(router); err != nil {
			return err
		}
	}
	return nil
}

func (c *ValidatorClient) initializeForWeb(cliCtx *cli.Context, router *mux.Router) error {
	dataDir := cliCtx.String(cmd.DataDirFlag.Name)
	dataFile := filepath.Join(dataDir, kv.ProtectionDbFileName)
	walletDir := cliCtx.String(flags.WalletDirFlag.Name)
	isInteropNumValidatorsSet := cliCtx.IsSet(flags.InteropNumValidators.Name)
	isWeb3SignerURLFlagSet := cliCtx.IsSet(flags.Web3SignerURLFlag.Name)

	if cliCtx.IsSet(flags.Web3SignerURLFlag.Name) {
		// Custom Check For Web3Signer
		c.wallet = wallet.NewWalletForWeb3Signer()
	} else {
		// Read the wallet password file from the cli context.
		if err := setWalletPasswordFilePath(cliCtx); err != nil {
			return errors.Wrap(err, "could not read wallet password file")
		}

		// Read the wallet from the specified path.
		w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
			return nil, nil
		})
		if err != nil {
			return errors.Wrap(err, "could not open wallet")
		}
		c.wallet = w
	}

	// Workaround for https://github.com/prysmaticlabs/prysm/issues/13391
	dataDir, _ = c.getLegacyDatabaseLocation(
		isInteropNumValidatorsSet,
		isWeb3SignerURLFlagSet,
		dataDir,
		dataFile,
		walletDir,
	)

	clearFlag := cliCtx.Bool(cmd.ClearDB.Name)
	forceClearFlag := cliCtx.Bool(cmd.ForceClearDB.Name)

	if clearFlag || forceClearFlag {
		if err := clearDB(cliCtx.Context, dataDir, forceClearFlag); err != nil {
			return err
		}
	}
	log.WithField("databasePath", dataDir).Info("Checking DB")
	valDB, err := kv.NewKVStore(cliCtx.Context, dataDir, &kv.Config{
		PubKeys: nil,
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize db")
	}
	c.db = valDB
	if err := valDB.RunUpMigrations(cliCtx.Context); err != nil {
		return errors.Wrap(err, "could not run database migration")
	}

	if !cliCtx.Bool(cmd.DisableMonitoringFlag.Name) {
		if err := c.registerPrometheusService(cliCtx); err != nil {
			return err
		}
	}
	if err := c.registerValidatorService(cliCtx); err != nil {
		return err
	}

	if err := c.registerRPCService(router); err != nil {
		return err
	}
	if err := c.registerRPCGatewayService(router); err != nil {
		return err
	}
	gatewayHost := cliCtx.String(flags.GRPCGatewayHost.Name)
	gatewayPort := cliCtx.Int(flags.GRPCGatewayPort.Name)
	webAddress := fmt.Sprintf("http://%s:%d", gatewayHost, gatewayPort)
	log.WithField("address", webAddress).Info(
		"Starting Prysm web UI on address, open in browser to access",
	)
	return nil
}

func (c *ValidatorClient) registerPrometheusService(cliCtx *cli.Context) error {
	var additionalHandlers []prometheus.Handler
	if cliCtx.IsSet(cmd.EnableBackupWebhookFlag.Name) {
		additionalHandlers = append(
			additionalHandlers,
			prometheus.Handler{
				Path:    "/db/backup",
				Handler: backup.Handler(c.db, cliCtx.String(cmd.BackupWebhookOutputDir.Name)),
			},
		)
	}
	service := prometheus.NewService(
		fmt.Sprintf("%s:%d", c.cliCtx.String(cmd.MonitoringHostFlag.Name), c.cliCtx.Int(flags.MonitoringPortFlag.Name)),
		c.services,
		additionalHandlers...,
	)
	logrus.AddHook(prometheus.NewLogrusCollector())
	return c.services.RegisterService(service)
}

func (c *ValidatorClient) registerValidatorService(cliCtx *cli.Context) error {
	var (
		endpoint             string        = c.cliCtx.String(flags.BeaconRPCProviderFlag.Name)
		dataDir              string        = c.cliCtx.String(cmd.DataDirFlag.Name)
		logValidatorBalances bool          = !c.cliCtx.Bool(flags.DisablePenaltyRewardLogFlag.Name)
		emitAccountMetrics   bool          = !c.cliCtx.Bool(flags.DisableAccountMetricsFlag.Name)
		cert                 string        = c.cliCtx.String(flags.CertFlag.Name)
		graffiti             string        = c.cliCtx.String(flags.GraffitiFlag.Name)
		maxCallRecvMsgSize   int           = c.cliCtx.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name)
		grpcRetries          uint          = c.cliCtx.Uint(flags.GrpcRetriesFlag.Name)
		grpcRetryDelay       time.Duration = c.cliCtx.Duration(flags.GrpcRetryDelayFlag.Name)

		interopKeysConfig *local.InteropKeymanagerConfig
		err               error
	)

	// Configure interop.
	if c.cliCtx.IsSet(flags.InteropNumValidators.Name) {
		interopKeysConfig = &local.InteropKeymanagerConfig{
			Offset:           cliCtx.Uint64(flags.InteropStartIndex.Name),
			NumValidatorKeys: cliCtx.Uint64(flags.InteropNumValidators.Name),
		}
	}

	// Configure graffiti.
	graffitiStruct := &g.Graffiti{}
	if c.cliCtx.IsSet(flags.GraffitiFileFlag.Name) {
		graffitiFilePath := c.cliCtx.String(flags.GraffitiFileFlag.Name)

		graffitiStruct, err = g.ParseGraffitiFile(graffitiFilePath)
		if err != nil {
			log.WithError(err).Warn("Could not parse graffiti file")
		}
	}

	web3signerConfig, err := Web3SignerConfig(c.cliCtx)
	if err != nil {
		return err
	}

	ps, err := proposerSettings(c.cliCtx, c.db)
	if err != nil {
		return err
	}

	validatorService, err := client.NewValidatorService(c.cliCtx.Context, &client.Config{
		Endpoint:                   endpoint,
		DataDir:                    dataDir,
		LogValidatorBalances:       logValidatorBalances,
		EmitAccountMetrics:         emitAccountMetrics,
		CertFlag:                   cert,
		GraffitiFlag:               g.ParseHexGraffiti(graffiti),
		GrpcMaxCallRecvMsgSizeFlag: maxCallRecvMsgSize,
		GrpcRetriesFlag:            grpcRetries,
		GrpcRetryDelay:             grpcRetryDelay,
		GrpcHeadersFlag:            c.cliCtx.String(flags.GrpcHeadersFlag.Name),
		ValDB:                      c.db,
		UseWeb:                     c.cliCtx.Bool(flags.EnableWebFlag.Name),
		InteropKeysConfig:          interopKeysConfig,
		Wallet:                     c.wallet,
		WalletInitializedFeed:      c.walletInitialized,
		GraffitiStruct:             graffitiStruct,
		Web3SignerConfig:           web3signerConfig,
		ProposerSettings:           ps,
		BeaconApiTimeout:           time.Second * 30,
		BeaconApiEndpoint:          c.cliCtx.String(flags.BeaconRESTApiProviderFlag.Name),
		ValidatorsRegBatchSize:     c.cliCtx.Int(flags.ValidatorsRegistrationBatchSizeFlag.Name),
		Distributed:                c.cliCtx.Bool(flags.EnableDistributed.Name),
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize validator service")
	}

	return c.services.RegisterService(validatorService)
}

func Web3SignerConfig(cliCtx *cli.Context) (*remoteweb3signer.SetupConfig, error) {
	var web3signerConfig *remoteweb3signer.SetupConfig
	if cliCtx.IsSet(flags.Web3SignerURLFlag.Name) {
		urlStr := cliCtx.String(flags.Web3SignerURLFlag.Name)
		u, err := url.ParseRequestURI(urlStr)
		if err != nil {
			return nil, errors.Wrapf(err, "web3signer url %s is invalid", urlStr)
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("web3signer url must be in the format of http(s)://host:port url used: %v", urlStr)
		}
		web3signerConfig = &remoteweb3signer.SetupConfig{
			BaseEndpoint:          u.String(),
			GenesisValidatorsRoot: nil,
		}
		if cliCtx.IsSet(flags.WalletPasswordFileFlag.Name) {
			log.Warnf("%s was provided while using web3signer and will be ignored", flags.WalletPasswordFileFlag.Name)
		}

		if publicKeysSlice := cliCtx.StringSlice(flags.Web3SignerPublicValidatorKeysFlag.Name); len(publicKeysSlice) > 0 {
			pks := make([]string, 0)
			if len(publicKeysSlice) == 1 {
				pURL, err := url.ParseRequestURI(publicKeysSlice[0])
				if err == nil && pURL.Scheme != "" && pURL.Host != "" {
					web3signerConfig.PublicKeysURL = publicKeysSlice[0]
				} else {
					pks = strings.Split(publicKeysSlice[0], ",")
				}
			} else if len(publicKeysSlice) > 1 {
				pks = publicKeysSlice
			}
			if len(pks) > 0 {
				pks = slice.Unique[string](pks)
				var validatorKeys [][48]byte
				for _, key := range pks {
					decodedKey, decodeErr := hexutil.Decode(key)
					if decodeErr != nil {
						return nil, errors.Wrapf(decodeErr, "could not decode public key for web3signer: %s", key)
					}
					validatorKeys = append(validatorKeys, bytesutil.ToBytes48(decodedKey))
				}
				web3signerConfig.ProvidedPublicKeys = validatorKeys
			}
		}
	}
	return web3signerConfig, nil
}

func proposerSettings(cliCtx *cli.Context, db iface.ValidatorDB) (*proposer.Settings, error) {
	l, err := loader.NewProposerSettingsLoader(
		cliCtx,
		db,
		loader.WithBuilderConfig(),
		loader.WithGasLimit(),
	)
	if err != nil {
		return nil, err
	}
	return l.Load(cliCtx)
}

func (c *ValidatorClient) registerRPCService(router *mux.Router) error {
	var vs *client.ValidatorService
	if err := c.services.FetchService(&vs); err != nil {
		return err
	}
	validatorGatewayHost := c.cliCtx.String(flags.GRPCGatewayHost.Name)
	validatorGatewayPort := c.cliCtx.Int(flags.GRPCGatewayPort.Name)
	validatorMonitoringHost := c.cliCtx.String(cmd.MonitoringHostFlag.Name)
	validatorMonitoringPort := c.cliCtx.Int(flags.MonitoringPortFlag.Name)
	rpcHost := c.cliCtx.String(flags.RPCHost.Name)
	rpcPort := c.cliCtx.Int(flags.RPCPort.Name)
	nodeGatewayEndpoint := c.cliCtx.String(flags.BeaconRPCGatewayProviderFlag.Name)
	beaconClientEndpoint := c.cliCtx.String(flags.BeaconRPCProviderFlag.Name)
	maxCallRecvMsgSize := c.cliCtx.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name)
	grpcRetries := c.cliCtx.Uint(flags.GrpcRetriesFlag.Name)
	grpcRetryDelay := c.cliCtx.Duration(flags.GrpcRetryDelayFlag.Name)
	walletDir := c.cliCtx.String(flags.WalletDirFlag.Name)
	grpcHeaders := c.cliCtx.String(flags.GrpcHeadersFlag.Name)
	clientCert := c.cliCtx.String(flags.CertFlag.Name)
	server := rpc.NewServer(c.cliCtx.Context, &rpc.Config{
		ValDB:                    c.db,
		Host:                     rpcHost,
		Port:                     fmt.Sprintf("%d", rpcPort),
		WalletInitializedFeed:    c.walletInitialized,
		ValidatorService:         vs,
		SyncChecker:              vs,
		GenesisFetcher:           vs,
		NodeGatewayEndpoint:      nodeGatewayEndpoint,
		WalletDir:                walletDir,
		Wallet:                   c.wallet,
		ValidatorGatewayHost:     validatorGatewayHost,
		ValidatorGatewayPort:     validatorGatewayPort,
		ValidatorMonitoringHost:  validatorMonitoringHost,
		ValidatorMonitoringPort:  validatorMonitoringPort,
		BeaconClientEndpoint:     beaconClientEndpoint,
		ClientMaxCallRecvMsgSize: maxCallRecvMsgSize,
		ClientGrpcRetries:        grpcRetries,
		ClientGrpcRetryDelay:     grpcRetryDelay,
		ClientGrpcHeaders:        strings.Split(grpcHeaders, ","),
		ClientWithCert:           clientCert,
		Router:                   router,
	})
	return c.services.RegisterService(server)
}

func (c *ValidatorClient) registerRPCGatewayService(router *mux.Router) error {
	gatewayHost := c.cliCtx.String(flags.GRPCGatewayHost.Name)
	if gatewayHost != flags.DefaultGatewayHost {
		log.WithField("webHost", gatewayHost).Warn(
			"You are using a non-default web host. Web traffic is served by HTTP, so be wary of " +
				"changing this parameter if you are exposing this host to the Internet!",
		)
	}
	gatewayPort := c.cliCtx.Int(flags.GRPCGatewayPort.Name)
	rpcHost := c.cliCtx.String(flags.RPCHost.Name)
	rpcPort := c.cliCtx.Int(flags.RPCPort.Name)
	rpcAddr := net.JoinHostPort(rpcHost, fmt.Sprintf("%d", rpcPort))
	gatewayAddress := net.JoinHostPort(gatewayHost, fmt.Sprintf("%d", gatewayPort))
	timeout := c.cliCtx.Int(cmd.ApiTimeoutFlag.Name)
	var allowedOrigins []string
	if c.cliCtx.IsSet(flags.GPRCGatewayCorsDomain.Name) {
		allowedOrigins = strings.Split(c.cliCtx.String(flags.GPRCGatewayCorsDomain.Name), ",")
	} else {
		allowedOrigins = strings.Split(flags.GPRCGatewayCorsDomain.Value, ",")
	}
	maxCallSize := c.cliCtx.Uint64(cmd.GrpcMaxCallRecvMsgSizeFlag.Name)

	registrations := []gateway.PbHandlerRegistration{
		pb.RegisterHealthHandler,
	}
	gwmux := gwruntime.NewServeMux(
		gwruntime.WithMarshalerOption(gwruntime.MIMEWildcard, &gwruntime.HTTPBodyMarshaler{
			Marshaler: &gwruntime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					EmitUnpopulated: true,
					UseProtoNames:   true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		}),
		gwruntime.WithMarshalerOption(
			api.EventStreamMediaType, &gwruntime.EventSourceJSONPb{}, // TODO: remove this
		),
		gwruntime.WithForwardResponseOption(gateway.HttpResponseModifier),
	)

	muxHandler := func(h http.HandlerFunc, w http.ResponseWriter, req *http.Request) {
		// The validator gateway handler requires this special logic as it serves the web APIs and the web UI.
		if strings.HasPrefix(req.URL.Path, "/api") {
			req.URL.Path = strings.Replace(req.URL.Path, "/api", "", 1)
			// Else, we handle with the Prysm API gateway without a middleware.
			h(w, req)
		} else {
			// Finally, we handle with the web server.
			web.Handler(w, req)
		}
	}

	pbHandler := &gateway.PbMux{
		Registrations: registrations,
		Mux:           gwmux,
	}
	opts := []gateway.Option{
		gateway.WithMuxHandler(muxHandler),
		gateway.WithRouter(router), // note some routes are registered in server.go
		gateway.WithRemoteAddr(rpcAddr),
		gateway.WithGatewayAddr(gatewayAddress),
		gateway.WithMaxCallRecvMsgSize(maxCallSize),
		gateway.WithPbHandlers([]*gateway.PbMux{pbHandler}),
		gateway.WithAllowedOrigins(allowedOrigins),
		gateway.WithTimeout(uint64(timeout)),
	}
	gw, err := gateway.New(c.cliCtx.Context, opts...)
	if err != nil {
		return err
	}
	return c.services.RegisterService(gw)
}

func setWalletPasswordFilePath(cliCtx *cli.Context) error {
	walletDir := cliCtx.String(flags.WalletDirFlag.Name)
	defaultWalletPasswordFilePath := filepath.Join(walletDir, wallet.DefaultWalletPasswordFile)
	if file.Exists(defaultWalletPasswordFilePath) {
		// Ensure file has proper permissions.
		hasPerms, err := file.HasReadWritePermissions(defaultWalletPasswordFilePath)
		if err != nil {
			return err
		}
		if !hasPerms {
			return fmt.Errorf(
				"wallet password file %s does not have proper 0600 permissions",
				defaultWalletPasswordFilePath,
			)
		}

		// Set the filepath into the cli context.
		if err := cliCtx.Set(flags.WalletPasswordFileFlag.Name, defaultWalletPasswordFilePath); err != nil {
			return errors.Wrap(err, "could not set default wallet password file path")
		}
	}
	return nil
}

func clearDB(ctx context.Context, dataDir string, force bool) error {
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
		valDB, err := kv.NewKVStore(ctx, dataDir, &kv.Config{})
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

func configureFastSSZHashingAlgorithm() {
	fastssz.EnableVectorizedHTR = true
}

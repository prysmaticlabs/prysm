// Package node is the main process which handles the lifecycle of
// the runtime services in a validator client process, gracefully shutting
// everything down upon close.
package node

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/shared/version"
	accountsv2 "github.com/prysmaticlabs/prysm/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/validator/client"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v1 "github.com/prysmaticlabs/prysm/validator/keymanager/v1"
	v2 "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	slashing_protection "github.com/prysmaticlabs/prysm/validator/slashing-protection"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "node")

// ValidatorClient defines an instance of an eth2 validator that manages
// the entire lifecycle of services attached to it participating in eth2.
type ValidatorClient struct {
	cliCtx   *cli.Context
	services *shared.ServiceRegistry // Lifecycle and service store.
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications.
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

	registry := shared.NewServiceRegistry()
	ValidatorClient := &ValidatorClient{
		cliCtx:   cliCtx,
		services: registry,
		stop:     make(chan struct{}),
	}

	featureconfig.ConfigureValidator(cliCtx)
	cmd.ConfigureValidator(cliCtx)

	if cliCtx.IsSet(cmd.ChainConfigFileFlag.Name) {
		chainConfigFileName := cliCtx.String(cmd.ChainConfigFileFlag.Name)
		params.LoadChainConfigFile(chainConfigFileName)
	}

	var keyManagerV1 v1.KeyManager
	var keyManagerV2 v2.IKeymanager
	if featureconfig.Get().EnableAccountsV2 {
		// Read the wallet from the specified path.
		wallet, err := accountsv2.OpenWallet(cliCtx)
		if err != nil {
			log.Fatalf("Could not open wallet: %v", err)
		}
		keyManagerV2, err = wallet.InitializeKeymanager(
			context.Background(), false, /* skipMnemonicConfirm */
		)
		if err != nil {
			log.Fatalf("Could not read existing keymanager for wallet: %v", err)
		}
	} else {
		keyManagerV1, err = selectV1Keymanager(cliCtx)
		if err != nil {
			return nil, err
		}
	}

	pubKeys, err := ExtractPublicKeysFromKeymanager(cliCtx, keyManagerV1, keyManagerV2)
	if err != nil {
		return nil, err
	}
	if len(pubKeys) == 0 {
		log.Warn("No keys found; nothing to validate")
	} else {
		log.WithField("validators", len(pubKeys)).Debug("Found validator keys")
		for _, key := range pubKeys {
			log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(key[:]))).Info("Validating for public key")
		}
	}

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
		if err := clearDB(dataDir, pubKeys, forceClearFlag); err != nil {
			return nil, err
		}
	}
	log.WithField("databasePath", dataDir).Info("Checking DB")

	if err := ValidatorClient.registerPrometheusService(); err != nil {
		return nil, err
	}
	if featureconfig.Get().SlasherProtection {
		if err := ValidatorClient.registerSlasherClientService(); err != nil {
			return nil, err
		}
	}
	if err := ValidatorClient.registerClientService(keyManagerV1, keyManagerV2, pubKeys); err != nil {
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

	close(s.stop)
}

func (s *ValidatorClient) registerPrometheusService() error {
	service := prometheus.NewPrometheusService(
		fmt.Sprintf("%s:%d", s.cliCtx.String(cmd.MonitoringHostFlag.Name), s.cliCtx.Int(flags.MonitoringPortFlag.Name)),
		s.services,
	)
	logrus.AddHook(prometheus.NewLogrusCollector())
	return s.services.RegisterService(service)
}

func (s *ValidatorClient) registerClientService(
	keyManager v1.KeyManager,
	keyManagerV2 v2.IKeymanager,
	validatingPubKeys [][48]byte,
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
	v, err := client.NewValidatorService(context.Background(), &client.Config{
		Endpoint:                   endpoint,
		DataDir:                    dataDir,
		KeyManager:                 keyManager,
		KeyManagerV2:               keyManagerV2,
		LogValidatorBalances:       logValidatorBalances,
		EmitAccountMetrics:         emitAccountMetrics,
		CertFlag:                   cert,
		GraffitiFlag:               graffiti,
		ValidatingPubKeys:          validatingPubKeys,
		GrpcMaxCallRecvMsgSizeFlag: maxCallRecvMsgSize,
		GrpcRetriesFlag:            grpcRetries,
		GrpcRetryDelay:             grpcRetryDelay,
		GrpcHeadersFlag:            s.cliCtx.String(flags.GrpcHeadersFlag.Name),
		Protector:                  protector,
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
	sp, err := slashing_protection.NewSlashingProtectionService(context.Background(), &slashing_protection.Config{
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

// Selects the key manager depending on the options provided by the user.
func selectV1Keymanager(ctx *cli.Context) (v1.KeyManager, error) {
	manager := strings.ToLower(ctx.String(flags.KeyManager.Name))
	opts := ctx.String(flags.KeyManagerOpts.Name)
	if opts == "" {
		opts = "{}"
	} else if !strings.HasPrefix(opts, "{") {
		fileopts, err := ioutil.ReadFile(opts)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read keymanager options file")
		}
		opts = string(fileopts)
	}

	if manager == "" {
		// Attempt to work out keymanager from deprecated vars.
		if unencryptedKeys := ctx.String(flags.UnencryptedKeysFlag.Name); unencryptedKeys != "" {
			manager = "unencrypted"
			opts = fmt.Sprintf(`{"path":%q}`, unencryptedKeys)
			log.Warn(fmt.Sprintf("--unencrypted-keys flag is deprecated.  Please use --keymanager=unencrypted --keymanageropts='%s'", opts))
		} else if numValidatorKeys := ctx.Uint64(flags.InteropNumValidators.Name); numValidatorKeys > 0 {
			manager = "interop"
			opts = fmt.Sprintf(`{"keys":%d,"offset":%d}`, numValidatorKeys, ctx.Uint64(flags.InteropStartIndex.Name))
			log.Warn(fmt.Sprintf("--interop-num-validators and --interop-start-index flags are deprecated.  Please use --keymanager=interop --keymanageropts='%s'", opts))
		} else if keystorePath := ctx.String(flags.KeystorePathFlag.Name); keystorePath != "" {
			manager = "keystore"
			opts = fmt.Sprintf(`{"path":%q,"passphrase":%q}`, keystorePath, ctx.String(flags.PasswordFlag.Name))
			log.Warn(fmt.Sprintf("--keystore-path flag is deprecated.  Please use --keymanager=keystore --keymanageropts='%s'", opts))
		} else {
			// Default if no choice made
			manager = "keystore"
			passphrase := ctx.String(flags.PasswordFlag.Name)
			if passphrase == "" {
				log.Warn("Implicit selection of keymanager is deprecated.  Please use --keymanager=keystore or select a different keymanager")
			} else {
				opts = fmt.Sprintf(`{"passphrase":%q}`, passphrase)
				log.Warn(`Implicit selection of keymanager is deprecated.  Please use --keymanager=keystore --keymanageropts='{"passphrase":"<password>"}' or select a different keymanager`)
			}
		}
	}

	var km v1.KeyManager
	var help string
	var err error
	switch manager {
	case "interop":
		km, help, err = v1.NewInterop(opts)
	case "unencrypted":
		km, help, err = v1.NewUnencrypted(opts)
	case "keystore":
		km, help, err = v1.NewKeystore(opts)
	case "wallet":
		km, help, err = v1.NewWallet(opts)
	case "remote":
		km, help, err = v1.NewRemoteWallet(opts)
	default:
		return nil, fmt.Errorf("unknown keymanager %q", manager)
	}
	if err != nil {
		if help != "" {
			// Print help for the keymanager
			fmt.Println(help)
		}
		return nil, err
	}
	return km, nil
}

func clearDB(dataDir string, pubkeys [][48]byte, force bool) error {
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
		valDB, err := kv.NewKVStore(dataDir, pubkeys)
		if err != nil {
			return errors.Wrapf(err, "Could not create DB in dir %s", dataDir)
		}

		log.Warning("Removing database")
		if err := valDB.ClearDB(); err != nil {
			return errors.Wrapf(err, "Could not clear DB in dir %s", dataDir)
		}
	}

	return nil
}

// ExtractPublicKeysFromKeymanager extracts only the public keys from the specified key manager.
func ExtractPublicKeysFromKeymanager(cliCtx *cli.Context, keyManagerV1 v1.KeyManager, keyManagerV2 v2.IKeymanager) ([][48]byte, error) {
	var pubKeys [][48]byte
	var err error
	if featureconfig.Get().EnableAccountsV2 {
		pubKeys, err = keyManagerV2.FetchValidatingPublicKeys(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain public keys for validation")
		}
		return pubKeys, nil
	}
	return keyManagerV1.FetchValidatingKeys()
}

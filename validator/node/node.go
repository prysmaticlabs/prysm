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
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/prysmaticlabs/prysm/shared/tracing"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/validator/client"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

var log = logrus.WithField("prefix", "node")

// ValidatorClient defines an instance of a sharding validator that manages
// the entire lifecycle of services attached to it participating in
// Ethereum Serenity.
type ValidatorClient struct {
	ctx      *cli.Context
	services *shared.ServiceRegistry // Lifecycle and service store.
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications.
}

// NewValidatorClient creates a new, Ethereum Serenity validator client.
func NewValidatorClient(ctx *cli.Context) (*ValidatorClient, error) {
	if err := tracing.Setup(
		"validator", // service name
		ctx.String(cmd.TracingProcessNameFlag.Name),
		ctx.String(cmd.TracingEndpointFlag.Name),
		ctx.Float64(cmd.TraceSampleFractionFlag.Name),
		ctx.Bool(cmd.EnableTracingFlag.Name),
	); err != nil {
		return nil, err
	}

	verbosity := ctx.String(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return nil, err
	}
	logrus.SetLevel(level)

	registry := shared.NewServiceRegistry()
	ValidatorClient := &ValidatorClient{
		ctx:      ctx,
		services: registry,
		stop:     make(chan struct{}),
	}

	if ctx.IsSet(cmd.ChainConfigFileFlag.Name) {
		chainConfigFileName := ctx.String(cmd.ChainConfigFileFlag.Name)
		params.LoadChainConfigFile(chainConfigFileName)
	}

	featureconfig.ConfigureValidator(ctx)

	keyManager, err := selectKeyManager(ctx)
	if err != nil {
		return nil, err
	}

	pubKeys, err := keyManager.FetchValidatingKeys()
	if err != nil {
		log.WithError(err).Error("Failed to obtain public keys for validation")
	} else {
		if len(pubKeys) == 0 {
			log.Warn("No keys found; nothing to validate")
		} else {
			log.WithField("validators", len(pubKeys)).Debug("Found validator keys")
			for _, key := range pubKeys {
				log.WithField("pubKey", fmt.Sprintf("%#x", key)).Info("Validating for public key")
			}
		}
	}

	clearFlag := ctx.Bool(cmd.ClearDB.Name)
	forceClearFlag := ctx.Bool(cmd.ForceClearDB.Name)
	dataDir := ctx.String(cmd.DataDirFlag.Name)
	if clearFlag || forceClearFlag {
		pubkeys, err := keyManager.FetchValidatingKeys()
		if err != nil {
			return nil, err
		}
		if dataDir == "" {
			dataDir = cmd.DefaultDataDir()
		}
		if err := clearDB(dataDir, pubkeys, forceClearFlag); err != nil {
			return nil, err
		}
	}
	log.WithField("databasePath", dataDir).Info("Checking DB")

	if err := ValidatorClient.registerPrometheusService(ctx); err != nil {
		return nil, err
	}

	if err := ValidatorClient.registerClientService(ctx, keyManager); err != nil {
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
		debug.Exit(s.ctx) // Ensure trace and CPU profile data are flushed.
		go s.Close()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Info("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
		panic("Panic closing the sharding validator")
	}()

	// Wait for stop channel to be closed.
	<-stop
}

// Close handles graceful shutdown of the system.
func (s *ValidatorClient) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.services.StopAll()
	log.Info("Stopping sharding validator")

	close(s.stop)
}

func (s *ValidatorClient) registerPrometheusService(ctx *cli.Context) error {
	service := prometheus.NewPrometheusService(
		fmt.Sprintf(":%d", ctx.Int64(flags.MonitoringPortFlag.Name)),
		s.services,
	)
	logrus.AddHook(prometheus.NewLogrusCollector())
	return s.services.RegisterService(service)
}

func (s *ValidatorClient) registerClientService(ctx *cli.Context, keyManager keymanager.KeyManager) error {
	endpoint := ctx.String(flags.BeaconRPCProviderFlag.Name)
	dataDir := ctx.String(cmd.DataDirFlag.Name)
	logValidatorBalances := !ctx.Bool(flags.DisablePenaltyRewardLogFlag.Name)
	emitAccountMetrics := ctx.Bool(flags.AccountMetricsFlag.Name)
	cert := ctx.String(flags.CertFlag.Name)
	graffiti := ctx.String(flags.GraffitiFlag.Name)
	maxCallRecvMsgSize := ctx.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name)
	grpcRetries := ctx.Uint(flags.GrpcRetriesFlag.Name)
	v, err := client.NewValidatorService(context.Background(), &client.Config{
		Endpoint:                   endpoint,
		DataDir:                    dataDir,
		KeyManager:                 keyManager,
		LogValidatorBalances:       logValidatorBalances,
		EmitAccountMetrics:         emitAccountMetrics,
		CertFlag:                   cert,
		GraffitiFlag:               graffiti,
		GrpcMaxCallRecvMsgSizeFlag: maxCallRecvMsgSize,
		GrpcRetriesFlag:            grpcRetries,
		GrpcHeadersFlag:            ctx.String(flags.GrpcHeadersFlag.Name),
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize client service")
	}
	return s.services.RegisterService(v)
}

// selectKeyManager selects the key manager depending on the options provided by the user.
func selectKeyManager(ctx *cli.Context) (keymanager.KeyManager, error) {
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

	var km keymanager.KeyManager
	var help string
	var err error
	switch manager {
	case "interop":
		km, help, err = keymanager.NewInterop(opts)
	case "unencrypted":
		km, help, err = keymanager.NewUnencrypted(opts)
	case "keystore":
		km, help, err = keymanager.NewKeystore(opts)
	case "wallet":
		km, help, err = keymanager.NewWallet(opts)
	case "remote":
		km, help, err = keymanager.NewRemoteWallet(opts)
	default:
		return nil, fmt.Errorf("unknown keymanager %q", manager)
	}
	if err != nil {
		// Print help for the keymanager
		fmt.Println(help)
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
			return errors.Wrapf(err, "Could not create DB in dir %s", dataDir)
		}
	}

	if clearDBConfirmed {
		valDB, err := db.NewKVStore(dataDir, pubkeys)
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

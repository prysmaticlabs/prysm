/*
Package features defines which features are enabled for runtime
in order to selectively enable certain features to maintain a stable runtime.

The process for implementing new features using this package is as follows:
	1. Add a new CMD flag in flags.go, and place it in the proper list(s) var for its client.
	2. Add a condition for the flag in the proper Configure function(s) below.
	3. Place any "new" behavior in the `if flagEnabled` statement.
	4. Place any "previous" behavior in the `else` statement.
	5. Ensure any tests using the new feature fail if the flag isn't enabled.
	5a. Use the following to enable your flag for tests:
	cfg := &featureconfig.Flags{
		VerifyAttestationSigs: true,
	}
	resetCfg := featureconfig.InitWithReset(cfg)
	defer resetCfg()
	6. Add the string for the flags that should be running within E2E to E2EValidatorFlags
	and E2EBeaconChainFlags.
*/
package features

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prysmaticlabs/gohashtree"
	"github.com/prysmaticlabs/prysm/v3/cmd"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "flags")

const enabledFeatureFlag = "Enabled feature flag"
const disabledFeatureFlag = "Disabled feature flag"

// Flags is a struct to represent which features the client will perform on runtime.
type Flags struct {
	// Feature related flags.
	RemoteSlasherProtection             bool // RemoteSlasherProtection utilizes a beacon node with --slasher mode for validator slashing protection.
	WriteSSZStateTransitions            bool // WriteSSZStateTransitions to tmp directory.
	EnablePeerScorer                    bool // EnablePeerScorer enables experimental peer scoring in p2p.
	WriteWalletPasswordOnWebOnboarding  bool // WriteWalletPasswordOnWebOnboarding writes the password to disk after Prysm web signup.
	EnableDoppelGanger                  bool // EnableDoppelGanger enables doppelganger protection on startup for the validator.
	EnableHistoricalSpaceRepresentation bool // EnableHistoricalSpaceRepresentation enables the saving of registry validators in separate buckets to save space
	// Logging related toggles.
	DisableGRPCConnectionLogs bool // Disables logging when a new grpc client has connected.

	// Slasher toggles.
	DisableBroadcastSlashings bool // DisableBroadcastSlashings disables p2p broadcasting of proposer and attester slashings.

	// Bug fixes related flags.
	AttestTimely bool // AttestTimely fixes #8185. It is gated behind a flag to ensure beacon node's fix can safely roll out first. We'll invert this in v1.1.0.

	EnableSlasher bool // Enable slasher in the beacon node runtime.
	// EnableSlashingProtectionPruning for the validator client.
	EnableSlashingProtectionPruning bool

	EnableNativeState                 bool // EnableNativeState defines whether the beacon state will be represented as a pure Go struct or a Go struct that wraps a proto struct.
	DisablePullTips                   bool // DisablePullTips disables experimental disabling of boundary checks.
	EnableDefensivePull               bool // EnableDefensivePull enables exerimental back boundary checks.
	EnableVectorizedHTR               bool // EnableVectorizedHTR specifies whether the beacon state will use the optimized sha256 routines.
	DisableForkchoiceDoublyLinkedTree bool // DisableForkChoiceDoublyLinkedTree specifies whether fork choice store will use a doubly linked tree.
	EnableBatchGossipAggregation      bool // EnableBatchGossipAggregation specifies whether to further aggregate our gossip batches before verifying them.
	EnableOnlyBlindedBeaconBlocks     bool // EnableOnlyBlindedBeaconBlocks enables only storing blinded beacon blocks in the DB post-Bellatrix fork.
	EnableStartOptimistic             bool // EnableStartOptimistic treats every block as optimistic at startup.

	// KeystoreImportDebounceInterval specifies the time duration the validator waits to reload new keys if they have
	// changed on disk. This feature is for advanced use cases only.
	KeystoreImportDebounceInterval time.Duration
}

var featureConfig *Flags
var featureConfigLock sync.RWMutex

// Get retrieves feature config.
func Get() *Flags {
	featureConfigLock.RLock()
	defer featureConfigLock.RUnlock()

	if featureConfig == nil {
		return &Flags{}
	}
	return featureConfig
}

// Init sets the global config equal to the config that is passed in.
func Init(c *Flags) {
	featureConfigLock.Lock()
	defer featureConfigLock.Unlock()

	featureConfig = c
}

// InitWithReset sets the global config and returns function that is used to reset configuration.
func InitWithReset(c *Flags) func() {
	var prevConfig Flags
	if featureConfig != nil {
		prevConfig = *featureConfig
	} else {
		prevConfig = Flags{}
	}
	resetFunc := func() {
		Init(&prevConfig)
	}
	Init(c)
	return resetFunc
}

// configureTestnet sets the config according to specified testnet flag
func configureTestnet(ctx *cli.Context) error {
	if ctx.Bool(PraterTestnet.Name) {
		log.Warn("Running on the Prater Testnet")
		if err := params.SetActive(params.PraterConfig().Copy()); err != nil {
			return err
		}
		applyPraterFeatureFlags(ctx)
		params.UsePraterNetworkConfig()
	} else if ctx.Bool(RopstenTestnet.Name) {
		log.Warn("Running on the Ropsten Beacon Chain Testnet")
		if err := params.SetActive(params.RopstenConfig().Copy()); err != nil {
			return err
		}
		applyRopstenFeatureFlags(ctx)
		params.UseRopstenNetworkConfig()
	} else if ctx.Bool(SepoliaTestnet.Name) {
		log.Warn("Running on the Sepolia Beacon Chain Testnet")
		if err := params.SetActive(params.SepoliaConfig().Copy()); err != nil {
			return err
		}
		applySepoliaFeatureFlags(ctx)
		params.UseSepoliaNetworkConfig()
	} else {
		if ctx.IsSet(cmd.ChainConfigFileFlag.Name) {
			log.Warn("Running on custom Ethereum network specified in a chain configuration yaml file")
		} else {
			log.Warn("Running on Ethereum Mainnet")
		}
		if err := params.SetActive(params.MainnetConfig().Copy()); err != nil {
			return err
		}
	}
	return nil
}

// Insert feature flags within the function to be enabled for Prater testnet.
func applyPraterFeatureFlags(ctx *cli.Context) {
	if err := ctx.Set(EnableOnlyBlindedBeaconBlocks.Names()[0], "true"); err != nil {
		log.WithError(err).Debug("error enabling only saving blinded beacon blocks flag")
	}
}

// Insert feature flags within the function to be enabled for Ropsten testnet.
func applyRopstenFeatureFlags(ctx *cli.Context) {
}

// Insert feature flags within the function to be enabled for Sepolia testnet.
func applySepoliaFeatureFlags(ctx *cli.Context) {
}

// ConfigureBeaconChain sets the global config based
// on what flags are enabled for the beacon-chain client.
func ConfigureBeaconChain(ctx *cli.Context) error {
	complainOnDeprecatedFlags(ctx)
	cfg := &Flags{}
	if ctx.Bool(devModeFlag.Name) {
		enableDevModeFlags(ctx)
	}
	if err := configureTestnet(ctx); err != nil {
		return err
	}

	if ctx.Bool(writeSSZStateTransitionsFlag.Name) {
		logEnabled(writeSSZStateTransitionsFlag)
		cfg.WriteSSZStateTransitions = true
	}

	if ctx.IsSet(disableGRPCConnectionLogging.Name) {
		logDisabled(disableGRPCConnectionLogging)
		cfg.DisableGRPCConnectionLogs = true
	}
	cfg.EnablePeerScorer = true
	if ctx.Bool(disablePeerScorer.Name) {
		logDisabled(disablePeerScorer)
		cfg.EnablePeerScorer = false
	}
	if ctx.Bool(disableBroadcastSlashingFlag.Name) {
		logDisabled(disableBroadcastSlashingFlag)
		cfg.DisableBroadcastSlashings = true
	}
	if ctx.Bool(enableSlasherFlag.Name) {
		log.WithField(enableSlasherFlag.Name, enableSlasherFlag.Usage).Warn(enabledFeatureFlag)
		cfg.EnableSlasher = true
	}
	if ctx.Bool(enableHistoricalSpaceRepresentation.Name) {
		log.WithField(enableHistoricalSpaceRepresentation.Name, enableHistoricalSpaceRepresentation.Usage).Warn(enabledFeatureFlag)
		cfg.EnableHistoricalSpaceRepresentation = true
	}
	cfg.EnableNativeState = true
	if ctx.Bool(disableNativeState.Name) {
		logDisabled(disableNativeState)
		cfg.EnableNativeState = false
	}

	if ctx.Bool(disablePullTips.Name) {
		logEnabled(disablePullTips)
		cfg.DisablePullTips = true
	}
	if ctx.Bool(enableDefensivePull.Name) {
		logEnabled(enableDefensivePull)
		cfg.EnableDefensivePull = true
	}

	if ctx.Bool(disableVecHTR.Name) {
		logEnabled(disableVecHTR)
	} else {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGILL)
		defer signal.Stop(sigc)
		buffer := make([][32]byte, 2)
		err := gohashtree.Hash(buffer, buffer)
		if err != nil {
			log.Error("could not test if gohashtree is supported")
		} else {
			t := time.NewTimer(time.Millisecond * 100)
			select {
			case <-sigc:
				log.Error("gohashtree is not supported in this CPU")
			case <-t.C:
				cfg.EnableVectorizedHTR = true
			}
		}
	}
	if ctx.Bool(disableForkChoiceDoublyLinkedTree.Name) {
		logEnabled(disableForkChoiceDoublyLinkedTree)
		cfg.DisableForkchoiceDoublyLinkedTree = true
	}
	cfg.EnableBatchGossipAggregation = true
	if ctx.Bool(disableGossipBatchAggregation.Name) {
		logDisabled(disableGossipBatchAggregation)
		cfg.EnableBatchGossipAggregation = false
	}
	if ctx.Bool(EnableOnlyBlindedBeaconBlocks.Name) {
		logEnabled(EnableOnlyBlindedBeaconBlocks)
		cfg.EnableOnlyBlindedBeaconBlocks = true
	}
	if ctx.Bool(enableStartupOptimistic.Name) {
		logEnabled(enableStartupOptimistic)
		cfg.EnableStartOptimistic = true
	}
	Init(cfg)
	return nil
}

// ConfigureValidator sets the global config based
// on what flags are enabled for the validator client.
func ConfigureValidator(ctx *cli.Context) error {
	complainOnDeprecatedFlags(ctx)
	cfg := &Flags{}
	if err := configureTestnet(ctx); err != nil {
		return err
	}
	if ctx.Bool(enableExternalSlasherProtectionFlag.Name) {
		log.Fatal(
			"Remote slashing protection has currently been disabled in Prysm due to safety concerns. " +
				"We appreciate your understanding in our desire to keep Prysm validators safe.",
		)
	}
	if ctx.Bool(writeWalletPasswordOnWebOnboarding.Name) {
		logEnabled(writeWalletPasswordOnWebOnboarding)
		cfg.WriteWalletPasswordOnWebOnboarding = true
	}
	if ctx.Bool(attestTimely.Name) {
		logEnabled(attestTimely)
		cfg.AttestTimely = true
	}
	if ctx.Bool(enableSlashingProtectionPruning.Name) {
		logEnabled(enableSlashingProtectionPruning)
		cfg.EnableSlashingProtectionPruning = true
	}
	if ctx.Bool(enableDoppelGangerProtection.Name) {
		logEnabled(enableDoppelGangerProtection)
		cfg.EnableDoppelGanger = true
	}
	cfg.KeystoreImportDebounceInterval = ctx.Duration(dynamicKeyReloadDebounceInterval.Name)
	Init(cfg)
	return nil
}

// enableDevModeFlags switches development mode features on.
func enableDevModeFlags(ctx *cli.Context) {
	log.Warn("Enabling development mode flags")
	for _, f := range devModeFlags {
		log.WithField("flag", f.Names()[0]).Debug("Enabling development mode flag")
		if !ctx.IsSet(f.Names()[0]) {
			if err := ctx.Set(f.Names()[0], "true"); err != nil {
				log.WithError(err).Debug("Error enabling development mode flag")
			}
		}
	}
}

func complainOnDeprecatedFlags(ctx *cli.Context) {
	for _, f := range deprecatedFlags {
		if ctx.IsSet(f.Names()[0]) {
			log.Errorf("%s is deprecated and has no effect. Do not use this flag, it will be deleted soon.", f.Names()[0])
		}
	}
}

func logEnabled(flag cli.DocGenerationFlag) {
	var name string
	if names := flag.Names(); len(names) > 0 {
		name = names[0]
	}
	log.WithField(name, flag.GetUsage()).Warn(enabledFeatureFlag)
}

func logDisabled(flag cli.DocGenerationFlag) {
	var name string
	if names := flag.Names(); len(names) > 0 {
		name = names[0]
	}
	log.WithField(name, flag.GetUsage()).Warn(disabledFeatureFlag)
}

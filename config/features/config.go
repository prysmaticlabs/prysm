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
	"sync"
	"time"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "flags")

const enabledFeatureFlag = "Enabled feature flag"
const disabledFeatureFlag = "Disabled feature flag"

// Flags is a struct to represent which features the client will perform on runtime.
type Flags struct {
	// Testnet Flags.
	PyrmontTestnet bool // PyrmontTestnet defines the flag through which we can enable the node to run on the Pyrmont testnet.

	// Feature related flags.
	RemoteSlasherProtection             bool // RemoteSlasherProtection utilizes a beacon node with --slasher mode for validator slashing protection.
	WriteSSZStateTransitions            bool // WriteSSZStateTransitions to tmp directory.
	SkipBLSVerify                       bool // Skips BLS verification across the runtime.
	EnablePeerScorer                    bool // EnablePeerScorer enables experimental peer scoring in p2p.
	EnableLargerGossipHistory           bool // EnableLargerGossipHistory increases the gossip history we store in our caches.
	WriteWalletPasswordOnWebOnboarding  bool // WriteWalletPasswordOnWebOnboarding writes the password to disk after Prysm web signup.
	DisableAttestingHistoryDBCache      bool // DisableAttestingHistoryDBCache for the validator client increases disk reads/writes.
	ProposerAttsSelectionUsingMaxCover  bool // ProposerAttsSelectionUsingMaxCover enables max-cover algorithm when selecting attestations for proposing.
	EnableOptimizedBalanceUpdate        bool // EnableOptimizedBalanceUpdate uses an updated method of performing balance updates.
	EnableDoppelGanger                  bool // EnableDoppelGanger enables doppelganger protection on startup for the validator.
	EnableHistoricalSpaceRepresentation bool // EnableHistoricalSpaceRepresentation enables the saving of registry validators in separate buckets to save space
	EnableGetBlockOptimizations         bool // EnableGetBlockOptimizations optimizes some elements of the GetBlock() function.
	EnableBatchVerification             bool // EnableBatchVerification enables batch signature verification on gossip messages.
	EnableBalanceTrieComputation        bool // EnableBalanceTrieComputation enables our beacon state to use balance tries for hash tree root operations.
	// Logging related toggles.
	DisableGRPCConnectionLogs bool // Disables logging when a new grpc client has connected.

	// Slasher toggles.
	DisableLookback           bool // DisableLookback updates slasher to not use the lookback and update validator histories until epoch 0.
	DisableBroadcastSlashings bool // DisableBroadcastSlashings disables p2p broadcasting of proposer and attester slashings.

	// Cache toggles.
	EnableSSZCache           bool // EnableSSZCache see https://github.com/prysmaticlabs/prysm/pull/4558.
	EnableNextSlotStateCache bool // EnableNextSlotStateCache enables next slot state cache to improve validator performance.
	EnableActiveBalanceCache bool // EnableActiveBalanceCache enables active balance cache.

	// Bug fixes related flags.
	AttestTimely bool // AttestTimely fixes #8185. It is gated behind a flag to ensure beacon node's fix can safely roll out first. We'll invert this in v1.1.0.

	EnableSlasher bool // Enable slasher in the beacon node runtime.
	// EnableSlashingProtectionPruning for the validator client.
	EnableSlashingProtectionPruning bool

	// Bug fixes related flags.
	CorrectlyInsertOrphanedAtts bool
	CorrectlyPruneCanonicalAtts bool

	// KeystoreImportDebounceInterval specifies the time duration the validator waits to reload new keys if they have
	// changed on disk. This feature is for advanced use cases only.
	KeystoreImportDebounceInterval time.Duration

	AttestationAggregationStrategy string // AttestationAggregationStrategy defines aggregation strategy to be used when aggregating.
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
func configureTestnet(ctx *cli.Context, cfg *Flags) {
	if ctx.Bool(PyrmontTestnet.Name) {
		log.Warn("Running on Pyrmont Testnet")
		params.UsePyrmontConfig()
		params.UsePyrmontNetworkConfig()
		cfg.PyrmontTestnet = true
	} else if ctx.Bool(PraterTestnet.Name) {
		log.Warn("Running on the Prater Testnet")
		params.UsePraterConfig()
		params.UsePraterNetworkConfig()
	} else {
		log.Warn("Running on Ethereum Consensus Mainnet")
		params.UseMainnetConfig()
	}
}

// ConfigureBeaconChain sets the global config based
// on what flags are enabled for the beacon-chain client.
func ConfigureBeaconChain(ctx *cli.Context) {
	complainOnDeprecatedFlags(ctx)
	cfg := &Flags{}
	if ctx.Bool(devModeFlag.Name) {
		enableDevModeFlags(ctx)
	}
	configureTestnet(ctx, cfg)

	if ctx.Bool(writeSSZStateTransitionsFlag.Name) {
		logEnabled(writeSSZStateTransitionsFlag)
		cfg.WriteSSZStateTransitions = true
	}

	cfg.EnableSSZCache = true

	if ctx.IsSet(disableGRPCConnectionLogging.Name) {
		logDisabled(disableGRPCConnectionLogging)
		cfg.DisableGRPCConnectionLogs = true
	}
	cfg.AttestationAggregationStrategy = ctx.String(attestationAggregationStrategy.Name)
	if ctx.Bool(forceOptMaxCoverAggregationStategy.Name) {
		logEnabled(forceOptMaxCoverAggregationStategy)
		cfg.AttestationAggregationStrategy = "opt_max_cover"
	}
	if ctx.Bool(enablePeerScorer.Name) {
		logEnabled(enablePeerScorer)
		cfg.EnablePeerScorer = true
	}
	if ctx.Bool(checkPtInfoCache.Name) {
		log.Warn("Advance check point info cache is no longer supported and will soon be deleted")
	}
	if ctx.Bool(enableLargerGossipHistory.Name) {
		logEnabled(enableLargerGossipHistory)
		cfg.EnableLargerGossipHistory = true
	}
	if ctx.Bool(disableBroadcastSlashingFlag.Name) {
		logDisabled(disableBroadcastSlashingFlag)
		cfg.DisableBroadcastSlashings = true
	}
	cfg.EnableNextSlotStateCache = true
	if ctx.Bool(disableNextSlotStateCache.Name) {
		logDisabled(disableNextSlotStateCache)
		cfg.EnableNextSlotStateCache = false
	}
	if ctx.Bool(enableSlasherFlag.Name) {
		log.WithField(enableSlasherFlag.Name, enableSlasherFlag.Usage).Warn(enabledFeatureFlag)
		cfg.EnableSlasher = true
	}
	cfg.ProposerAttsSelectionUsingMaxCover = true
	if ctx.Bool(disableProposerAttsSelectionUsingMaxCover.Name) {
		logDisabled(disableProposerAttsSelectionUsingMaxCover)
		cfg.ProposerAttsSelectionUsingMaxCover = false
	}
	cfg.EnableOptimizedBalanceUpdate = true
	if ctx.Bool(disableOptimizedBalanceUpdate.Name) {
		logDisabled(disableOptimizedBalanceUpdate)
		cfg.EnableOptimizedBalanceUpdate = false
	}
	if ctx.Bool(enableHistoricalSpaceRepresentation.Name) {
		log.WithField(enableHistoricalSpaceRepresentation.Name, enableHistoricalSpaceRepresentation.Usage).Warn(enabledFeatureFlag)
		cfg.EnableHistoricalSpaceRepresentation = true
	}
	cfg.CorrectlyInsertOrphanedAtts = true
	if ctx.Bool(disableCorrectlyInsertOrphanedAtts.Name) {
		logDisabled(disableCorrectlyInsertOrphanedAtts)
		cfg.CorrectlyInsertOrphanedAtts = false
	}
	cfg.CorrectlyPruneCanonicalAtts = true
	if ctx.Bool(disableCorrectlyPruneCanonicalAtts.Name) {
		logDisabled(disableCorrectlyPruneCanonicalAtts)
		cfg.CorrectlyPruneCanonicalAtts = false
	}
	cfg.EnableActiveBalanceCache = true
	if ctx.Bool(disableActiveBalanceCache.Name) {
		logDisabled(disableActiveBalanceCache)
		cfg.EnableActiveBalanceCache = false
	}
	if ctx.Bool(enableGetBlockOptimizations.Name) {
		logEnabled(enableGetBlockOptimizations)
		cfg.EnableGetBlockOptimizations = true
	}
	if ctx.Bool(enableBatchGossipVerification.Name) {
		logEnabled(enableBatchGossipVerification)
		cfg.EnableBatchVerification = true
	}
	if ctx.Bool(enableBalanceTrieComputation.Name) {
		logEnabled(enableBalanceTrieComputation)
		cfg.EnableBalanceTrieComputation = true
	}
	Init(cfg)
}

// ConfigureValidator sets the global config based
// on what flags are enabled for the validator client.
func ConfigureValidator(ctx *cli.Context) {
	complainOnDeprecatedFlags(ctx)
	cfg := &Flags{}
	configureTestnet(ctx, cfg)
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
	if ctx.Bool(disableAttestingHistoryDBCache.Name) {
		logDisabled(disableAttestingHistoryDBCache)
		cfg.DisableAttestingHistoryDBCache = true
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

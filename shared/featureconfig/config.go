/*
Package featureconfig defines which features are enabled for runtime
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
package featureconfig

import (
	"sync"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "flags")

// Flags is a struct to represent which features the client will perform on runtime.
type Flags struct {
	// Testnet Flags.
	ToledoTestnet  bool // ToledoTestnet defines the flag through which we can enable the node to run on the Toledo testnet.
	PyrmontTestnet bool // PyrmontTestnet defines the flag through which we can enable the node to run on the Pyrmont testnet.

	// Feature related flags.
	WriteSSZStateTransitions           bool // WriteSSZStateTransitions to tmp directory.
	SkipBLSVerify                      bool // Skips BLS verification across the runtime.
	EnableBlst                         bool // Enables new BLS library from supranational.
	PruneEpochBoundaryStates           bool // PruneEpochBoundaryStates prunes the epoch boundary state before last finalized check point.
	EnableSnappyDBCompression          bool // EnableSnappyDBCompression in the database.
	SlasherProtection                  bool // SlasherProtection protects validator fron sending over a slashable offense over the network using external slasher.
	EnableNoise                        bool // EnableNoise enables the beacon node to use NOISE instead of SECIO when performing a handshake with another peer.
	EnableEth1DataMajorityVote         bool // EnableEth1DataMajorityVote uses the Voting With The Majority algorithm to vote for eth1data.
	EnablePeerScorer                   bool // EnablePeerScorer enables experimental peer scoring in p2p.
	EnablePruningDepositProofs         bool // EnablePruningDepositProofs enables pruning deposit proofs which significantly reduces the size of a deposit
	EnableSyncBacktracking             bool // EnableSyncBacktracking enables backtracking algorithm when searching for alternative forks during initial sync.
	EnableLargerGossipHistory          bool // EnableLargerGossipHistory increases the gossip history we store in our caches.
	WriteWalletPasswordOnWebOnboarding bool // WriteWalletPasswordOnWebOnboarding writes the password to disk after Prysm web signup.
	DisableAttestingHistoryDBCache     bool // DisableAttestingHistoryDBCache for the validator client increases disk reads/writes.

	// Logging related toggles.
	DisableGRPCConnectionLogs bool // Disables logging when a new grpc client has connected.

	// Slasher toggles.
	EnableHistoricalDetection bool // EnableHistoricalDetection disables historical attestation detection and performs detection on the chain head immediately.
	DisableLookback           bool // DisableLookback updates slasher to not use the lookback and update validator histories until epoch 0.
	DisableBroadcastSlashings bool // DisableBroadcastSlashings disables p2p broadcasting of proposer and attester slashings.

	// Cache toggles.
	EnableSSZCache           bool // EnableSSZCache see https://github.com/prysmaticlabs/prysm/pull/4558.
	EnableEth1DataVoteCache  bool // EnableEth1DataVoteCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableSlasherConnection  bool // EnableSlasher enable retrieval of slashing events from a slasher instance.
	UseCheckPointInfoCache   bool // UseCheckPointInfoCache uses check point info cache to efficiently verify attestation signatures.
	EnableNextSlotStateCache bool // EnableNextSlotStateCache enables next slot state cache to improve validator performance.

	// Bug fixes related flags.
	AttestTimely bool // AttestTimely fixes #8185. It is gated behind a flag to ensure beacon node's fix can safely roll out first. We'll invert this in v1.1.0.

	KafkaBootstrapServers          string // KafkaBootstrapServers to find kafka servers to stream blocks, attestations, etc.
	AttestationAggregationStrategy string // AttestationAggregationStrategy defines aggregation strategy to be used when aggregating.

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
func configureTestnet(ctx *cli.Context, cfg *Flags) {
	if ctx.Bool(ToledoTestnet.Name) {
		log.Warn("Running on Toledo Testnet")
		params.UseToledoConfig()
		params.UseToledoNetworkConfig()
		cfg.ToledoTestnet = true
	} else if ctx.Bool(PyrmontTestnet.Name) {
		log.Warn("Running on Pyrmont Testnet")
		params.UsePyrmontConfig()
		params.UsePyrmontNetworkConfig()
		cfg.PyrmontTestnet = true
	} else {
		log.Warn("Running on ETH2 Mainnet")
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
		log.Warn("Writing SSZ states and blocks after state transitions")
		cfg.WriteSSZStateTransitions = true
	}

	cfg.EnableSSZCache = true

	if ctx.String(kafkaBootstrapServersFlag.Name) != "" {
		log.Warn("Enabling experimental kafka streaming.")
		cfg.KafkaBootstrapServers = ctx.String(kafkaBootstrapServersFlag.Name)
	}

	if ctx.IsSet(disableGRPCConnectionLogging.Name) {
		cfg.DisableGRPCConnectionLogs = true
	}
	cfg.AttestationAggregationStrategy = ctx.String(attestationAggregationStrategy.Name)
	log.Infof("Using %q strategy on attestation aggregation", cfg.AttestationAggregationStrategy)

	cfg.EnableEth1DataMajorityVote = true
	if ctx.Bool(disableEth1DataMajorityVote.Name) {
		log.Warn("Disabling eth1data majority vote")
		cfg.EnableEth1DataMajorityVote = false
	}
	if ctx.Bool(enablePeerScorer.Name) {
		log.Warn("Enabling peer scoring in P2P")
		cfg.EnablePeerScorer = true
	}
	if ctx.Bool(checkPtInfoCache.Name) {
		log.Warn("Advance check point info cache is no longer supported and will soon be deleted")
	}
	cfg.EnableBlst = true
	if ctx.Bool(disableBlst.Name) {
		log.Warn("Disabling new BLS library blst")
		cfg.EnableBlst = false
	}
	cfg.EnablePruningDepositProofs = true
	if ctx.Bool(disablePruningDepositProofs.Name) {
		log.Warn("Disabling pruning deposit proofs")
		cfg.EnablePruningDepositProofs = false
	}
	cfg.EnableSyncBacktracking = true
	if ctx.Bool(disableSyncBacktracking.Name) {
		log.Warn("Disabling init-sync backtracking algorithm")
		cfg.EnableSyncBacktracking = false
	}
	if ctx.Bool(enableLargerGossipHistory.Name) {
		log.Warn("Using a larger gossip history for the node")
		cfg.EnableLargerGossipHistory = true
	}
	if ctx.Bool(disableBroadcastSlashingFlag.Name) {
		log.Warn("Disabling slashing broadcasting to p2p network")
		cfg.DisableBroadcastSlashings = true
	}
	if ctx.Bool(enableNextSlotStateCache.Name) {
		log.Warn("Enabling next slot state cache")
		cfg.EnableNextSlotStateCache = true
	}
	Init(cfg)
}

// ConfigureSlasher sets the global config based
// on what flags are enabled for the slasher client.
func ConfigureSlasher(ctx *cli.Context) {
	complainOnDeprecatedFlags(ctx)
	cfg := &Flags{}
	configureTestnet(ctx, cfg)

	if ctx.Bool(disableLookbackFlag.Name) {
		log.Warn("Disabling slasher lookback")
		cfg.DisableLookback = true
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
		log.Warn("Enabled validator attestation and block slashing protection using an external slasher.")
		cfg.SlasherProtection = true
	}
	if ctx.Bool(writeWalletPasswordOnWebOnboarding.Name) {
		log.Warn("Enabled full web mode, wallet password will be written to disk at the wallet directory " +
			"upon completing web onboarding.")
		cfg.WriteWalletPasswordOnWebOnboarding = true
	}
	if ctx.Bool(disableAttestingHistoryDBCache.Name) {
		log.Warn("Disabled attesting history DB cache, likely increasing disk reads and writes significantly")
		cfg.DisableAttestingHistoryDBCache = true
	}
	cfg.EnableBlst = true
	if ctx.Bool(disableBlst.Name) {
		log.Warn("Disabling new BLS library blst")
		cfg.EnableBlst = false
	}
	if ctx.Bool(attestTimely.Name) {
		log.Warn("Enabled attest timely fix for #8185")
		cfg.AttestTimely = true
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

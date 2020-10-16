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

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "flags")

// Flags is a struct to represent which features the client will perform on runtime.
type Flags struct {
	// Testnet Flags.
	AltonaTestnet  bool // AltonaTestnet defines the flag through which we can enable the node to run on the Altona testnet.
	OnyxTestnet    bool // OnyxTestnet defines the flag through which we can enable the node to run on the Onyx testnet.
	MedallaTestnet bool // MedallaTestnet defines the flag through which we can enable the node to run on the Medalla testnet.
	SpadinaTestnet bool // SpadinaTestnet defines the flag through which we can enable the node to run on the Spadina testnet.
	ZinkenTestnet  bool // ZinkenTestnet defines the flag through which we can enable the node to run on the Zinken testnet.

	// Feature related flags.
	WriteSSZStateTransitions            bool // WriteSSZStateTransitions to tmp directory.
	SkipBLSVerify                       bool // Skips BLS verification across the runtime.
	EnableBlst                          bool // Enables new BLS library from supranational.
	EnableBackupWebhook                 bool // EnableBackupWebhook to allow database backups to trigger from monitoring port /db/backup.
	PruneEpochBoundaryStates            bool // PruneEpochBoundaryStates prunes the epoch boundary state before last finalized check point.
	EnableSnappyDBCompression           bool // EnableSnappyDBCompression in the database.
	SlasherProtection                   bool // SlasherProtection protects validator fron sending over a slashable offense over the network using external slasher.
	EnableNoise                         bool // EnableNoise enables the beacon node to use NOISE instead of SECIO when performing a handshake with another peer.
	WaitForSynced                       bool // WaitForSynced uses WaitForSynced in validator startup to ensure it can communicate with the beacon node as soon as possible.
	EnableEth1DataMajorityVote          bool // EnableEth1DataMajorityVote uses the Voting With The Majority algorithm to vote for eth1data.
	EnablePeerScorer                    bool // EnablePeerScorer enables experimental peer scoring in p2p.
	EnablePruningDepositProofs          bool // EnablePruningDepositProofs enables pruning deposit proofs which significantly reduces the size of a deposit
	EnableAttBroadcastDiscoveryAttempts bool // EnableAttBroadcastDiscoveryAttempts allows the p2p service to attempt to ensure a subnet peer is present before broadcasting an attestation.

	// Logging related toggles.
	DisableGRPCConnectionLogs bool // Disables logging when a new grpc client has connected.

	// Slasher toggles.
	EnableHistoricalDetection bool // EnableHistoricalDetection disables historical attestation detection and performs detection on the chain head immediately.
	DisableLookback           bool // DisableLookback updates slasher to not use the lookback and update validator histories until epoch 0.

	// Cache toggles.
	EnableSSZCache          bool // EnableSSZCache see https://github.com/prysmaticlabs/prysm/pull/4558.
	EnableEth1DataVoteCache bool // EnableEth1DataVoteCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableSlasherConnection bool // EnableSlasher enable retrieval of slashing events from a slasher instance.
	UseCheckPointInfoCache  bool // UseCheckPointInfoCache uses check point info cache to efficiently verify attestation signatures.

	KafkaBootstrapServers          string // KafkaBootstrapServers to find kafka servers to stream blocks, attestations, etc.
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
	resetFunc := func() {
		Init(&Flags{})
	}
	Init(c)
	return resetFunc
}

// configureTestnet sets the config according to specified testnet flag
func configureTestnet(ctx *cli.Context, cfg *Flags) {
	if ctx.Bool(AltonaTestnet.Name) {
		log.Warn("Running on Altona Testnet")
		params.UseAltonaConfig()
		params.UseAltonaNetworkConfig()
		cfg.AltonaTestnet = true
	} else if ctx.Bool(OnyxTestnet.Name) {
		log.Warn("Running on Onyx Testnet")
		params.UseOnyxConfig()
		params.UseOnyxNetworkConfig()
		cfg.OnyxTestnet = true
	} else if ctx.Bool(MedallaTestnet.Name) {
		log.Warn("Running on Medalla Testnet")
		params.UseMedallaConfig()
		params.UseMedallaNetworkConfig()
		cfg.MedallaTestnet = true
	} else if ctx.Bool(SpadinaTestnet.Name) {
		log.Warn("Running on Spadina Testnet")
		params.UseSpadinaConfig()
		params.UseSpadinaNetworkConfig()
		cfg.SpadinaTestnet = true
	} else if ctx.Bool(ZinkenTestnet.Name) {
		log.Warn("Running on Zinken Testnet")
		params.UseZinkenConfig()
		params.UseZinkenNetworkConfig()
		cfg.ZinkenTestnet = true
	} else {
		log.Warn("--<testnet> flag is not specified (default: Medalla), this will become required from next release! ")
		params.UseMedallaConfig()
		params.UseMedallaNetworkConfig()
		cfg.MedallaTestnet = true
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

	if ctx.Bool(enableBackupWebhookFlag.Name) {
		log.Warn("Allowing database backups to be triggered from HTTP webhook.")
		cfg.EnableBackupWebhook = true
	}
	if ctx.String(kafkaBootstrapServersFlag.Name) != "" {
		log.Warn("Enabling experimental kafka streaming.")
		cfg.KafkaBootstrapServers = ctx.String(kafkaBootstrapServersFlag.Name)
	}

	if ctx.IsSet(disableGRPCConnectionLogging.Name) {
		cfg.DisableGRPCConnectionLogs = true
	}
	cfg.AttestationAggregationStrategy = ctx.String(attestationAggregationStrategy.Name)
	log.Infof("Using %q strategy on attestation aggregation", cfg.AttestationAggregationStrategy)

	if ctx.Bool(enableEth1DataMajorityVote.Name) {
		log.Warn("Enabling eth1data majority vote")
		cfg.EnableEth1DataMajorityVote = true
	}
	if ctx.Bool(enableAttBroadcastDiscoveryAttempts.Name) {
		cfg.EnableAttBroadcastDiscoveryAttempts = true
	}
	if ctx.Bool(enablePeerScorer.Name) {
		log.Warn("Enabling peer scoring in P2P")
		cfg.EnablePeerScorer = true
	}
	if ctx.Bool(checkPtInfoCache.Name) {
		log.Warn("Using advance check point info cache")
		cfg.UseCheckPointInfoCache = true
	}
	if ctx.Bool(enableBlst.Name) {
		log.Warn("Enabling new BLS library blst")
		cfg.EnableBlst = true
	}
	if ctx.Bool(enablePruningDepositProofs.Name) {
		log.Warn("Enabling pruning deposit proofs")
		cfg.EnablePruningDepositProofs = true
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

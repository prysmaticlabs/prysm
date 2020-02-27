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
	featureconfig.Init(cfg)
	6. Add the string for the flags that should be running within E2E to E2EValidatorFlags
	and E2EBeaconChainFlags.
*/
package featureconfig

import (
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "flags")

// Flags is a struct to represent which features the client will perform on runtime.
type Flags struct {
	CustomGenesisDelay                         uint64 // CustomGenesisDelay signals how long of a delay to set to start the chain.
	MinimalConfig                              bool   // MinimalConfig as defined in the spec.
	WriteSSZStateTransitions                   bool   // WriteSSZStateTransitions to tmp directory.
	InitSyncNoVerify                           bool   // InitSyncNoVerify when initial syncing w/o verifying block's contents.
	SkipBLSVerify                              bool   // Skips BLS verification across the runtime.
	EnableBackupWebhook                        bool   // EnableBackupWebhook to allow database backups to trigger from monitoring port /db/backup.
	PruneEpochBoundaryStates                   bool   // PruneEpochBoundaryStates prunes the epoch boundary state before last finalized check point.
	EnableSnappyDBCompression                  bool   // EnableSnappyDBCompression in the database.
	KafkaBootstrapServers                      string // KafkaBootstrapServers to find kafka servers to stream blocks, attestations, etc.
	ProtectProposer                            bool   // ProtectProposer prevents the validator client from signing any proposals that would be considered a slashable offense.
	ProtectAttester                            bool   // ProtectAttester prevents the validator client from signing any attestations that would be considered a slashable offense.
	DisableStrictAttestationPubsubVerification bool   // DisableStrictAttestationPubsubVerification will disabling strict signature verification in pubsub.
	DisableUpdateHeadPerAttestation            bool   // DisableUpdateHeadPerAttestation will disabling update head on per attestation basis.
	EnableByteMempool                          bool   // EnaableByteMempool memory management.
	EnableDomainDataCache                      bool   // EnableDomainDataCache caches validator calls to DomainData per epoch.
	EnableStateGenSigVerify                    bool   // EnableStateGenSigVerify verifies proposer and randao signatures during state gen.

	// DisableForkChoice disables using LMD-GHOST fork choice to update
	// the head of the chain based on attestations and instead accepts any valid received block
	// as the chain head. UNSAFE, use with caution.
	DisableForkChoice bool

	// Cache toggles.
	EnableSSZCache          bool // EnableSSZCache see https://github.com/prysmaticlabs/prysm/pull/4558.
	EnableEth1DataVoteCache bool // EnableEth1DataVoteCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableSkipSlotsCache    bool // EnableSkipSlotsCache caches the state in skipped slots.
	EnableSlasherConnection bool // EnableSlasher enable retrieval of slashing events from a slasher instance.
	EnableBlockTreeCache    bool // EnableBlockTreeCache enable fork choice service to maintain latest filtered block tree.
}

var featureConfig *Flags

// Get retrieves feature config.
func Get() *Flags {
	if featureConfig == nil {
		return &Flags{}
	}
	return featureConfig
}

// Init sets the global config equal to the config that is passed in.
func Init(c *Flags) {
	featureConfig = c
}

// ConfigureBeaconChain sets the global config based
// on what flags are enabled for the beacon-chain client.
func ConfigureBeaconChain(ctx *cli.Context) {
	complainOnDeprecatedFlags(ctx)
	cfg := &Flags{}
	delay := ctx.GlobalUint64(customGenesisDelayFlag.Name)
	if delay != params.BeaconConfig().MinGenesisDelay {
		log.Warnf("Starting ETH2 with genesis delay of %d seconds", delay)
	}
	cfg.CustomGenesisDelay = delay
	if ctx.GlobalBool(minimalConfigFlag.Name) {
		log.Warn("Using minimal config")
		cfg.MinimalConfig = true
	}
	if ctx.GlobalBool(writeSSZStateTransitionsFlag.Name) {
		log.Warn("Writing SSZ states and blocks after state transitions")
		cfg.WriteSSZStateTransitions = true
	}
	if ctx.GlobalBool(disableForkChoiceUnsafeFlag.Name) {
		log.Warn("UNSAFE: Disabled fork choice for updating chain head")
		cfg.DisableForkChoice = true
	}
	if ctx.GlobalBool(enableSSZCache.Name) {
		log.Warn("Enabled unsafe ssz cache")
		cfg.EnableSSZCache = true
	}
	if ctx.GlobalBool(enableEth1DataVoteCacheFlag.Name) {
		log.Warn("Enabled unsafe eth1 data vote cache")
		cfg.EnableEth1DataVoteCache = true
	}
	if ctx.GlobalBool(initSyncVerifyEverythingFlag.Name) {
		log.Warn("Initial syncing with verifying all block's content signatures.")
		cfg.InitSyncNoVerify = false
	} else {
		cfg.InitSyncNoVerify = true
	}
	if ctx.GlobalBool(skipBLSVerifyFlag.Name) {
		log.Warn("UNSAFE: Skipping BLS verification at runtime")
		cfg.SkipBLSVerify = true
	}
	if ctx.GlobalBool(enableBackupWebhookFlag.Name) {
		log.Warn("Allowing database backups to be triggered from HTTP webhook.")
		cfg.EnableBackupWebhook = true
	}
	if ctx.GlobalBool(enableSkipSlotsCacheFlag.Name) {
		log.Warn("Enabled skip slots cache.")
		cfg.EnableSkipSlotsCache = true
	}
	if ctx.GlobalString(kafkaBootstrapServersFlag.Name) != "" {
		log.Warn("Enabling experimental kafka streaming.")
		cfg.KafkaBootstrapServers = ctx.GlobalString(kafkaBootstrapServersFlag.Name)
	}
	if ctx.GlobalBool(enableSlasherFlag.Name) {
		log.Warn("Enable slasher connection.")
		cfg.EnableSlasherConnection = true
	}
	if ctx.GlobalBool(cacheFilteredBlockTreeFlag.Name) {
		log.Warn("Enabled filtered block tree cache for fork choice.")
		cfg.EnableBlockTreeCache = true
	}
	if ctx.GlobalBool(disableStrictAttestationPubsubVerificationFlag.Name) {
		log.Warn("Disabled strict attestation signature verification in pubsub")
		cfg.DisableStrictAttestationPubsubVerification = true
	}
	if ctx.GlobalBool(disableUpdateHeadPerAttestation.Name) {
		log.Warn("Disabled update head on per attestation basis")
		cfg.DisableUpdateHeadPerAttestation = true
	}
	if ctx.GlobalBool(enableByteMempool.Name) {
		log.Warn("Enabling experimental memory management for beacon state")
		cfg.EnableByteMempool = true
	}
	if ctx.GlobalBool(enableStateGenSigVerify.Name) {
		log.Warn("Enabling sig verify for state gen")
		cfg.EnableStateGenSigVerify = true
	}
	Init(cfg)
}

// ConfigureValidator sets the global config based
// on what flags are enabled for the validator client.
func ConfigureValidator(ctx *cli.Context) {
	complainOnDeprecatedFlags(ctx)
	cfg := &Flags{}
	if ctx.GlobalBool(minimalConfigFlag.Name) {
		log.Warn("Using minimal config")
		cfg.MinimalConfig = true
	}
	if ctx.GlobalBool(protectProposerFlag.Name) {
		log.Warn("Enabled validator proposal slashing protection.")
		cfg.ProtectProposer = true
	}
	if ctx.GlobalBool(protectAttesterFlag.Name) {
		log.Warn("Enabled validator attestation slashing protection.")
		cfg.ProtectAttester = true
	}
	if ctx.GlobalBool(enableDomainDataCacheFlag.Name) {
		log.Warn("Enabled domain data cache.")
		cfg.EnableDomainDataCache = true
	}
	Init(cfg)
}

func complainOnDeprecatedFlags(ctx *cli.Context) {
	for _, f := range deprecatedFlags {
		if ctx.IsSet(f.GetName()) {
			log.Errorf("%s is deprecated and has no effect. Do not use this flag, it will be deleted soon.", f.GetName())
		}
	}
}

/*
Package featureconfig defines which features are enabled for runtime
in order to selctively enable certain features to maintain a stable runtime.

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
*/
package featureconfig

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "flags")

// Flags is a struct to represent which features the client will perform on runtime.
type Flags struct {
	GenesisDelay              bool // GenesisDelay when processing a chain start genesis event.
	MinimalConfig             bool // MinimalConfig as defined in the spec.
	WriteSSZStateTransitions  bool // WriteSSZStateTransitions to tmp directory.
	InitSyncNoVerify          bool // InitSyncNoVerify when initial syncing w/o verifying block's contents.
	SkipBLSVerify             bool // Skips BLS verification across the runtime.
	EnableBackupWebhook       bool // EnableBackupWebhook to allow database backups to trigger from monitoring port /db/backup.
	PruneEpochBoundaryStates  bool // PruneEpochBoundaryStates prunes the epoch boundary state before last finalized check point.
	EnableSnappyDBCompression bool // EnableSnappyDBCompression in the database.
	EnableCustomStateSSZ      bool // EnableCustomStateSSZ in the the state transition function.
	InitSyncCacheState        bool // InitSyncCacheState caches state during initial sync.

	// Cache toggles.
	EnableAttestationCache   bool // EnableAttestationCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableEth1DataVoteCache  bool // EnableEth1DataVoteCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableNewCache           bool // EnableNewCache enables the node to use the new caching scheme.
	EnableBLSPubkeyCache     bool // EnableBLSPubkeyCache to improve wall time of PubkeyFromBytes.
	EnableShuffledIndexCache bool // EnableShuffledIndexCache to cache expensive shuffled index computation.
	EnableSkipSlotsCache     bool // EnableSkipSlotsCache caches the state in skipped slots.
	EnableCommitteeCache     bool // EnableCommitteeCache to cache committee computation.
	EnableActiveIndicesCache bool // EnableActiveIndicesCache.
	EnableActiveCountCache   bool // EnableActiveCountCache.
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
	if ctx.GlobalBool(MinimalConfigFlag.Name) {
		log.Warn("Using minimal config")
		cfg.MinimalConfig = true
	}
	if ctx.GlobalBool(GenesisDelayFlag.Name) {
		log.Warn("Using non standard genesis delay. This may cause problems in a multi-node environment.")
		cfg.GenesisDelay = true
	}
	if ctx.GlobalBool(writeSSZStateTransitionsFlag.Name) {
		log.Warn("Writing SSZ states and blocks after state transitions")
		cfg.WriteSSZStateTransitions = true
	}
	if ctx.GlobalBool(EnableAttestationCacheFlag.Name) {
		log.Warn("Enabled unsafe attestation cache")
		cfg.EnableAttestationCache = true
	}
	if ctx.GlobalBool(EnableEth1DataVoteCacheFlag.Name) {
		log.Warn("Enabled unsafe eth1 data vote cache")
		cfg.EnableEth1DataVoteCache = true
	}
	if ctx.GlobalBool(EnableCustomStateSSZ.Name) {
		log.Warn("Enabled custom state ssz for the state transition function")
		cfg.EnableCustomStateSSZ = true
	}
	if ctx.GlobalBool(initSyncVerifyEverythingFlag.Name) {
		log.Warn("Initial syncing with verifying all block's content signatures.")
		cfg.InitSyncNoVerify = false
	} else {
		cfg.InitSyncNoVerify = true
	}
	if ctx.GlobalBool(NewCacheFlag.Name) {
		log.Warn("Using new cache for committee shuffled indices")
		cfg.EnableNewCache = true
	}
	if ctx.GlobalBool(SkipBLSVerifyFlag.Name) {
		log.Warn("UNSAFE: Skipping BLS verification at runtime")
		cfg.SkipBLSVerify = true
	}
	if ctx.GlobalBool(enableBackupWebhookFlag.Name) {
		log.Warn("Allowing database backups to be triggered from HTTP webhook.")
		cfg.EnableBackupWebhook = true
	}
	if ctx.GlobalBool(enableBLSPubkeyCacheFlag.Name) {
		log.Warn("Enabled BLS pubkey cache.")
		cfg.EnableBLSPubkeyCache = true
	}
	if ctx.GlobalBool(enableShuffledIndexCache.Name) {
		log.Warn("Enabled shuffled index cache.")
		cfg.EnableShuffledIndexCache = true
	}
	if ctx.GlobalBool(enableSkipSlotsCache.Name) {
		log.Warn("Enabled skip slots cache.")
		cfg.EnableSkipSlotsCache = true
	}
	if ctx.GlobalBool(enableCommitteeCacheFlag.Name) {
		log.Warn("Enabled committee cache.")
		cfg.EnableCommitteeCache = true
	}
	if ctx.GlobalBool(enableActiveIndicesCacheFlag.Name) {
		log.Warn("Enabled active indices cache.")
		cfg.EnableActiveIndicesCache = true
	}
	if ctx.GlobalBool(enableActiveCountCacheFlag.Name) {
		log.Warn("Enabled active count cache.")
		cfg.EnableActiveCountCache = true
	}
	if ctx.GlobalBool(enableSnappyDBCompressionFlag.Name) {
		log.Warn("Enabled snappy compression in the database.")
		cfg.EnableSnappyDBCompression = true
	}
	if ctx.GlobalBool(enablePruneBoundaryStateFlag.Name) {
		log.Warn("Enabled pruning epoch boundary states before last finalized check point.")
		cfg.PruneEpochBoundaryStates = true
	}
	if ctx.GlobalBool(initSyncCacheState.Name) {
		log.Warn("Enabled initial sync cache state mode.")
		cfg.InitSyncCacheState = true
	}
	Init(cfg)
}

// ConfigureValidator sets the global config based
// on what flags are enabled for the validator client.
func ConfigureValidator(ctx *cli.Context) {
	complainOnDeprecatedFlags(ctx)
	cfg := &Flags{}
	if ctx.GlobalBool(MinimalConfigFlag.Name) {
		log.Warn("Using minimal config")
		cfg.MinimalConfig = true
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

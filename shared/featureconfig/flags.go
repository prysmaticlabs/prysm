package featureconfig

import (
	"github.com/urfave/cli"
)

var (
	// GenesisDelayFlag disables the standard genesis delay.
	GenesisDelayFlag = cli.BoolFlag{
		Name:  "genesis-delay",
		Usage: "Wait and process the genesis event at the midnight of the next day rather than 30s after the ETH1 block time of the chainstart triggering deposit",
	}
	// MinimalConfigFlag enables the minimal configuration.
	MinimalConfigFlag = cli.BoolFlag{
		Name:  "minimal-config",
		Usage: "Use minimal config with parameters as defined in the spec.",
	}
	writeSSZStateTransitionsFlag = cli.BoolFlag{
		Name:  "interop-write-ssz-state-transitions",
		Usage: "Write ssz states to disk after attempted state transition",
	}
	// EnableAttestationCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableAttestationCacheFlag = cli.BoolFlag{
		Name:  "enable-attestation-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableEth1DataVoteCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableEth1DataVoteCacheFlag = cli.BoolFlag{
		Name:  "enable-eth1-data-vote-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableCustomStateSSZ see https://github.com/prysmaticlabs/prysm/pull/4077.
	EnableCustomStateSSZ = cli.BoolFlag{
		Name:  "enable-custom-state-ssz",
		Usage: "Enable custom hash_tree_root(state) for Prysm. See https://github.com/prysmaticlabs/prysm/issues/4077",
	}
	enableShuffledIndexCache = cli.BoolFlag{
		Name:  "enable-shuffled-index-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	enableCommitteeCacheFlag = cli.BoolFlag{
		Name:  "enable-committee-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	enableActiveIndicesCacheFlag = cli.BoolFlag{
		Name:  "enable-active-indices-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	enableActiveCountCacheFlag = cli.BoolFlag{
		Name:  "enable-active-count-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// NewCacheFlag enables the node to use the new caching scheme.
	NewCacheFlag = cli.BoolFlag{
		Name:  "new-cache",
		Usage: "Use the new shuffled indices cache for committee. Much improvement than previous caching implementations",
	}
	// SkipBLSVerifyFlag skips BLS signature verification across the runtime for development purposes.
	SkipBLSVerifyFlag = cli.BoolFlag{
		Name:  "skip-bls-verify",
		Usage: "Whether or not to skip BLS verification of signature at runtime, this is unsafe and should only be used for development",
	}
	enableBackupWebhookFlag = cli.BoolFlag{
		Name:  "enable-db-backup-webhook",
		Usage: "Serve HTTP handler to initiate database backups. The handler is served on the monitoring port at path /db/backup.",
	}
	enableBLSPubkeyCacheFlag = cli.BoolFlag{
		Name:  "enable-bls-pubkey-cache",
		Usage: "Enable BLS pubkey cache to improve wall time of PubkeyFromBytes",
	}
	// enableSkipSlotsCache enables the skips slots lru cache to be used in runtime.
	enableSkipSlotsCache = cli.BoolFlag{
		Name:  "enable-skip-slots-cache",
		Usage: "Enables the skip slot cache to be used in the event of skipped slots.",
	}
	enableSnappyDBCompressionFlag = cli.BoolFlag{
		Name:  "snappy",
		Usage: "Enables snappy compression in the database.",
	}
	enablePruneBoundaryStateFlag = cli.BoolFlag{
		Name:  "prune-states",
		Usage: "Prune epoch boundary states before last finalized check point",
	}
	initSyncVerifyEverythingFlag = cli.BoolFlag{
		Name: "initial-sync-verify-all-signatures",
		Usage: "Initial sync to finalized checkpoint with verifying block's signature, RANDAO " +
			"and attestation's aggregated signatures. Without this flag, only the proposer " +
			"signature is verified until the node reaches the end of the finalized chain.",
	}
	initSyncCacheState = cli.BoolFlag{
		Name: "initial-sync-cache-state",
		Usage: "Save state in cache during initial sync. We currently save state in the DB during " +
			"initial sync and disk-IO is one of the biggest bottleneck. This still saves finalized state in DB " +
			"and start syncing from there",
	}
)

// Deprecated flags list.
const deprecatedUsage = "DEPRECATED. DO NOT USE."

var (
	deprecatedNoGenesisDelayFlag = cli.BoolFlag{
		Name:   "no-genesis-delay",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableFinalizedBlockRootIndexFlag = cli.BoolFlag{
		Name:   "enable-finalized-block-root-index",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedOptimizeProcessEpoch = cli.BoolFlag{
		Name:   "optimize-process-epoch",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedPruneFinalizedStatesFlag = cli.BoolFlag{
		Name:   "prune-finalized-states",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedScatterFlag = cli.BoolFlag{
		Name:   "scatter",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedInitSyncNoVerifyFlag = cli.BoolFlag{
		Name:   "init-sync-no-verify",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

var deprecatedFlags = []cli.Flag{
	deprecatedNoGenesisDelayFlag,
	deprecatedEnableFinalizedBlockRootIndexFlag,
	deprecatedScatterFlag,
	deprecatedPruneFinalizedStatesFlag,
	deprecatedOptimizeProcessEpoch,
	deprecatedInitSyncNoVerifyFlag,
}

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = append(deprecatedFlags, []cli.Flag{
	MinimalConfigFlag,
}...)

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = append(deprecatedFlags, []cli.Flag{
	GenesisDelayFlag,
	MinimalConfigFlag,
	writeSSZStateTransitionsFlag,
	EnableAttestationCacheFlag,
	EnableEth1DataVoteCacheFlag,
	EnableCustomStateSSZ,
	initSyncVerifyEverythingFlag,
	initSyncCacheState,
	NewCacheFlag,
	SkipBLSVerifyFlag,
	enableBackupWebhookFlag,
	enableBLSPubkeyCacheFlag,
	enableShuffledIndexCache,
	enableCommitteeCacheFlag,
	enableActiveIndicesCacheFlag,
	enableActiveCountCacheFlag,
	enableSkipSlotsCache,
	enableSnappyDBCompressionFlag,
	enablePruneBoundaryStateFlag,
}...)

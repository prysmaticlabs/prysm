package featureconfig

import (
	"github.com/urfave/cli"
)

var (
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
	enableShuffledIndexCache = cli.BoolFlag{
		Name:  "enable-shuffled-index-cache",
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
	// enableSkipSlotsCache enables the skips slots lru cache to be used in runtime.
	enableSkipSlotsCache = cli.BoolFlag{
		Name:  "enable-skip-slots-cache",
		Usage: "Enables the skip slot cache to be used in the event of skipped slots.",
	}
	kafkaBootstrapServersFlag = cli.StringFlag{
		Name:  "kafka-url",
		Usage: "Stream attestations and blocks to specified kafka servers. This field is used for bootstrap.servers kafka config field.",
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
	saveDepositData = cli.BoolFlag{
		Name:  "save-deposit-data",
		Usage: "Enable of the saving of deposit related data",
	}
	noGenesisDelayFlag = cli.BoolFlag{
		Name: "no-genesis-delay",
		Usage: "Start the genesis event right away using the eth1 block timestamp which " +
			"triggered the genesis as the genesis time. This flag should be used for local " +
			"development and testing only.",
	}
)

// Deprecated flags list.
const deprecatedUsage = "DEPRECATED. DO NOT USE."

var (
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
	deprecatedEnableSnappyDBCompressionFlag = cli.BoolFlag{
		Name:   "snappy",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnablePruneBoundaryStateFlag = cli.BoolFlag{
		Name:   "prune-states",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableActiveIndicesCacheFlag = cli.BoolFlag{
		Name:   "enable-active-indices-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableActiveCountCacheFlag = cli.BoolFlag{
		Name:   "enable-active-count-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}

	deprecatedEnableCustomStateSSZ = cli.BoolFlag{
		Name:   "enable-custom-state-ssz",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableCommitteeCacheFlag = cli.BoolFlag{
		Name:   "enable-committee-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableBLSPubkeyCacheFlag = cli.BoolFlag{
		Name:   "enable-bls-pubkey-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedFastCommitteeAssignmentsFlag = cli.BoolFlag{
		Name:   "fast-assignments",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedGenesisDelayFlag = cli.BoolFlag{
		Name:   "genesis-delay",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

var deprecatedFlags = []cli.Flag{
	deprecatedEnableFinalizedBlockRootIndexFlag,
	deprecatedScatterFlag,
	deprecatedPruneFinalizedStatesFlag,
	deprecatedOptimizeProcessEpoch,
	deprecatedEnableSnappyDBCompressionFlag,
	deprecatedEnablePruneBoundaryStateFlag,
	deprecatedEnableActiveIndicesCacheFlag,
	deprecatedEnableActiveCountCacheFlag,
	deprecatedEnableCustomStateSSZ,
	deprecatedEnableCommitteeCacheFlag,
	deprecatedEnableBLSPubkeyCacheFlag,
	deprecatedFastCommitteeAssignmentsFlag,
	deprecatedGenesisDelayFlag,
}

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = append(deprecatedFlags, []cli.Flag{
	MinimalConfigFlag,
}...)

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = append(deprecatedFlags, []cli.Flag{
	noGenesisDelayFlag,
	MinimalConfigFlag,
	writeSSZStateTransitionsFlag,
	EnableAttestationCacheFlag,
	EnableEth1DataVoteCacheFlag,
	initSyncVerifyEverythingFlag,
	initSyncCacheState,
	NewCacheFlag,
	SkipBLSVerifyFlag,
	kafkaBootstrapServersFlag,
	enableBackupWebhookFlag,
	enableShuffledIndexCache,
	enableSkipSlotsCache,
	saveDepositData,
}...)

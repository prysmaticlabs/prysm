package featureconfig

import (
	"gopkg.in/urfave/cli.v2"
)

var (
	devModeFlag = &cli.BoolFlag{
		Name:  "dev",
		Usage: "Enable experimental features still in development. These features may not be stable.",
	}
	broadcastSlashingFlag = &cli.BoolFlag{
		Name:  "broadcast-slashing",
		Usage: "Broadcast slashings from slashing pool.",
	}
	minimalConfigFlag = &cli.BoolFlag{
		Name:  "minimal-config",
		Usage: "Use minimal config with parameters as defined in the spec.",
	}
	schlesiTestnetFlag = &cli.BoolFlag{
		Name:  "schlesi-testnet",
		Usage: "Use the preconfigured Schlesi multi-client testnet spec.",
	}
	writeSSZStateTransitionsFlag = &cli.BoolFlag{
		Name:  "interop-write-ssz-state-transitions",
		Usage: "Write ssz states to disk after attempted state transition",
	}
	disableDynamicCommitteeSubnets = &cli.BoolFlag{
		Name:  "disable-dynamic-committee-subnets",
		Usage: "Disable dynamic committee attestation subnets.",
	}
	// disableForkChoiceUnsafeFlag disables using the LMD-GHOST fork choice to update
	// the head of the chain based on attestations and instead accepts any valid received block
	// as the chain head. UNSAFE, use with caution.
	disableForkChoiceUnsafeFlag = &cli.BoolFlag{
		Name:  "disable-fork-choice-unsafe",
		Usage: "UNSAFE: disable fork choice for determining head of the beacon chain.",
	}
	// disableSSZCache see https://github.com/prysmaticlabs/prysm/pull/4558.
	disableSSZCache = &cli.BoolFlag{
		Name:  "disable-ssz-cache",
		Usage: "Disable ssz state root cache mechanism.",
	}
	// enableEth1DataVoteCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	enableEth1DataVoteCacheFlag = &cli.BoolFlag{
		Name:  "enable-eth1-data-vote-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	skipBLSVerifyFlag = &cli.BoolFlag{
		Name:  "skip-bls-verify",
		Usage: "Whether or not to skip BLS verification of signature at runtime, this is unsafe and should only be used for development",
	}
	enableBackupWebhookFlag = &cli.BoolFlag{
		Name:  "enable-db-backup-webhook",
		Usage: "Serve HTTP handler to initiate database backups. The handler is served on the monitoring port at path /db/backup.",
	}
	kafkaBootstrapServersFlag = &cli.StringFlag{
		Name:  "kafka-url",
		Usage: "Stream attestations and blocks to specified kafka servers. This field is used for bootstrap.servers kafka config field.",
	}
	initSyncVerifyEverythingFlag = &cli.BoolFlag{
		Name: "initial-sync-verify-all-signatures",
		Usage: "Initial sync to finalized checkpoint with verifying block's signature, RANDAO " +
			"and attestation's aggregated signatures. Without this flag, only the proposer " +
			"signature is verified until the node reaches the end of the finalized chain.",
	}
	enableSlasherFlag = &cli.BoolFlag{
		Name: "enable-slasher",
		Usage: "Enables connection to a slasher service in order to retrieve slashable events. Slasher is connected to the beacon node using gRPC and " +
			"the slasher-provider flag can be used to pass its address.",
	}
	customGenesisDelayFlag = &cli.Uint64Flag{
		Name: "custom-genesis-delay",
		Usage: "Start the genesis event with the configured genesis delay in seconds. " +
			"This flag should be used for local development and testing only.",
	}
	cacheFilteredBlockTreeFlag = &cli.BoolFlag{
		Name: "cache-filtered-block-tree",
		Usage: "Cache filtered block tree by maintaining it rather than continually recalculating on the fly, " +
			"this is used for fork choice.",
	}
	disableProtectProposerFlag = &cli.BoolFlag{
		Name: "disable-protect-proposer",
		Usage: "Disables functionality to prevent the validator client from signing and " +
			"broadcasting 2 different block proposals in the same epoch. Protects from slashing.",
		Value: true,
	}
	disableProtectAttesterFlag = &cli.BoolFlag{
		Name: "disable-protect-attester",
		Usage: "Disables functionality to prevent the validator client from signing and " +
			"broadcasting 2 any slashable attestations.",
		Value: true,
	}
	disableStrictAttestationPubsubVerificationFlag = &cli.BoolFlag{
		Name:  "disable-strict-attestation-pubsub-verification",
		Usage: "Disable strict signature verification of attestations in pubsub. See PR 4782 for details.",
	}
	disableUpdateHeadPerAttestation = &cli.BoolFlag{
		Name:  "disable-update-head-attestation",
		Usage: "Disable update fork choice head on per attestation. See PR 4802 for details.",
	}
	enableByteMempool = &cli.BoolFlag{
		Name:  "enable-byte-mempool",
		Usage: "Enable use of sync.Pool for certain byte arrays in the beacon state",
	}
	enableDomainDataCacheFlag = &cli.BoolFlag{
		Name: "enable-domain-data-cache",
		Usage: "Enable caching of domain data requests per epoch. This feature reduces the total " +
			"calls to the beacon node for each assignment.",
	}
	enableStateGenSigVerify = &cli.BoolFlag{
		Name: "enable-state-gen-sig-verify",
		Usage: "Enable signature verification for state gen. This feature increases the cost to generate a historical state," +
			"the resulting state is signature verified.",
	}
	checkHeadState = &cli.BoolFlag{
		Name:  "check-head-state",
		Usage: "Enables the checking of head state in chainservice first before retrieving the desired state from the db.",
	}
	enableNoiseHandshake = &cli.BoolFlag{
		Name: "enable-noise",
		Usage: "This enables the beacon node to use NOISE instead of SECIO for performing handshakes between peers and " +
			"securing transports between peers",
	}
	dontPruneStateStartUp = &cli.BoolFlag{
		Name:  "dont-prune-state-start-up",
		Usage: "Don't prune historical states upon start up",
	}
	enableNewStateMgmt = &cli.BoolFlag{
		Name:  "enable-new-state-mgmt",
		Usage: "This enable the usage of state mgmt service across Prysm",
	}
	enableFieldTrie = &cli.BoolFlag{
		Name:  "enable-state-field-trie",
		Usage: "Enables the usage of state field tries to compute the state root",
	}
	enableCustomBlockHTR = &cli.BoolFlag{
		Name:  "enable-custom-block-htr",
		Usage: "Enables the usage of a custom hashing method for our block",
	}
	disableInitSyncBatchSaveBlocks = &cli.BoolFlag{
		Name:  "disable-init-sync-batch-save-blocks",
		Usage: "Instead of saving batch blocks to the DB during initial syncing, this disables batch saving of blocks",
	}
	enableStateRefCopy = &cli.BoolFlag{
		Name:  "enable-state-ref-copy",
		Usage: "Enables the usage of a new copying method for our state fields.",
	}
	waitForSyncedFlag = &cli.BoolFlag{
		Name:  "wait-for-synced",
		Usage: "Uses WaitForSynced for validator startup, to ensure a validator is able to communicate with the beacon node as quick as possible",
	}
	disableHistoricalDetectionFlag = &cli.BoolFlag{
		Name:  "disable-historical-detection",
		Usage: "Disables historical attestation detection for the slasher",
	}
)

// devModeFlags holds list of flags that are set when development mode is on.
var devModeFlags = []cli.Flag{
	enableCustomBlockHTR,
	enableStateRefCopy,
	enableFieldTrie,
	enableNewStateMgmt,
}

// Deprecated flags list.
const deprecatedUsage = "DEPRECATED. DO NOT USE."

var (
	deprecatedEnableDynamicCommitteeSubnets = &cli.BoolFlag{
		Name:   "enable-dynamic-committee-subnets",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedNoCustomConfigFlag = &cli.BoolFlag{
		Name:   "no-custom-config",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableInitSyncQueue = &cli.BoolFlag{
		Name:   "enable-initial-sync-queue",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableFinalizedBlockRootIndexFlag = &cli.BoolFlag{
		Name:   "enable-finalized-block-root-index",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedOptimizeProcessEpochFlag = &cli.BoolFlag{
		Name:   "optimize-process-epoch",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedPruneFinalizedStatesFlag = &cli.BoolFlag{
		Name:   "prune-finalized-states",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedScatterFlag = &cli.BoolFlag{
		Name:   "scatter",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableSnappyDBCompressionFlag = &cli.BoolFlag{
		Name:   "snappy",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableSkipSlotsCacheFlag = &cli.BoolFlag{
		Name:   "enable-skip-slots-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnablePruneBoundaryStateFlag = &cli.BoolFlag{
		Name:   "prune-states",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableActiveIndicesCacheFlag = &cli.BoolFlag{
		Name:   "enable-active-indices-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableActiveCountCacheFlag = &cli.BoolFlag{
		Name:   "enable-active-count-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}

	deprecatedEnableCustomStateSSZFlag = &cli.BoolFlag{
		Name:   "enable-custom-state-ssz",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableCommitteeCacheFlag = &cli.BoolFlag{
		Name:   "enable-committee-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableBLSPubkeyCacheFlag = &cli.BoolFlag{
		Name:   "enable-bls-pubkey-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedFastCommitteeAssignmentsFlag = &cli.BoolFlag{
		Name:   "fast-assignments",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedGenesisDelayFlag = &cli.BoolFlag{
		Name:   "genesis-delay",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedNewCacheFlag = &cli.BoolFlag{
		Name:   "new-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableShuffledIndexCacheFlag = &cli.BoolFlag{
		Name:   "enable-shuffled-index-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedSaveDepositDataFlag = &cli.BoolFlag{
		Name:   "save-deposit-data",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedCacheProposerIndicesFlag = &cli.BoolFlag{
		Name:   "cache-proposer-indices",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedprotoArrayForkChoice = &cli.BoolFlag{
		Name:   "proto-array-forkchoice",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedForkchoiceAggregateAttestations = &cli.BoolFlag{
		Name:   "forkchoice-aggregate-attestations",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableAttestationCacheFlag = &cli.BoolFlag{
		Name:   "enable-attestation-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedInitSyncCacheStateFlag = &cli.BoolFlag{
		Name:   "initial-sync-cache-state",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedProtectProposerFlag = &cli.BoolFlag{
		Name:   "protect-proposer",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedProtectAttesterFlag = &cli.BoolFlag{
		Name:   "protect-attester",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDiscv5Flag = &cli.BoolFlag{
		Name:   "enable-discv5",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableSSZCache = &cli.BoolFlag{
		Name:   "enable-ssz-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedUseSpanCacheFlag = &cli.BoolFlag{
		Name:   "span-map-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableInitSyncQueueFlag = &cli.BoolFlag{
		Name:   "disable-init-sync-queue",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

var deprecatedFlags = []cli.Flag{
	deprecatedEnableDynamicCommitteeSubnets,
	deprecatedNoCustomConfigFlag,
	deprecatedEnableInitSyncQueue,
	deprecatedEnableFinalizedBlockRootIndexFlag,
	deprecatedScatterFlag,
	deprecatedPruneFinalizedStatesFlag,
	deprecatedOptimizeProcessEpochFlag,
	deprecatedEnableSnappyDBCompressionFlag,
	deprecatedEnableSkipSlotsCacheFlag,
	deprecatedEnablePruneBoundaryStateFlag,
	deprecatedEnableActiveIndicesCacheFlag,
	deprecatedEnableActiveCountCacheFlag,
	deprecatedEnableCustomStateSSZFlag,
	deprecatedEnableCommitteeCacheFlag,
	deprecatedEnableBLSPubkeyCacheFlag,
	deprecatedFastCommitteeAssignmentsFlag,
	deprecatedGenesisDelayFlag,
	deprecatedNewCacheFlag,
	deprecatedEnableShuffledIndexCacheFlag,
	deprecatedSaveDepositDataFlag,
	deprecatedCacheProposerIndicesFlag,
	deprecatedprotoArrayForkChoice,
	deprecatedForkchoiceAggregateAttestations,
	deprecatedEnableAttestationCacheFlag,
	deprecatedInitSyncCacheStateFlag,
	deprecatedProtectAttesterFlag,
	deprecatedProtectProposerFlag,
	deprecatedDiscv5Flag,
	deprecatedEnableSSZCache,
	deprecatedUseSpanCacheFlag,
	deprecatedDisableInitSyncQueueFlag,
}

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = append(deprecatedFlags, []cli.Flag{
	minimalConfigFlag,
	schlesiTestnetFlag,
	disableProtectAttesterFlag,
	disableProtectProposerFlag,
	enableDomainDataCacheFlag,
	waitForSyncedFlag,
}...)

// SlasherFlags contains a list of all the feature flags that apply to the slasher client.
var SlasherFlags = append(deprecatedFlags, []cli.Flag{
	disableHistoricalDetectionFlag,
}...)

// E2EValidatorFlags contains a list of the validator feature flags to be tested in E2E.
var E2EValidatorFlags = []string{
	"--enable-domain-data-cache",
	"--wait-for-synced",
}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = append(deprecatedFlags, []cli.Flag{
	devModeFlag,
	customGenesisDelayFlag,
	minimalConfigFlag,
	schlesiTestnetFlag,
	writeSSZStateTransitionsFlag,
	disableForkChoiceUnsafeFlag,
	disableDynamicCommitteeSubnets,
	disableSSZCache,
	enableEth1DataVoteCacheFlag,
	initSyncVerifyEverythingFlag,
	skipBLSVerifyFlag,
	kafkaBootstrapServersFlag,
	enableBackupWebhookFlag,
	enableSlasherFlag,
	cacheFilteredBlockTreeFlag,
	disableStrictAttestationPubsubVerificationFlag,
	disableUpdateHeadPerAttestation,
	enableByteMempool,
	enableStateGenSigVerify,
	checkHeadState,
	enableNoiseHandshake,
	dontPruneStateStartUp,
	broadcastSlashingFlag,
	enableNewStateMgmt,
	enableFieldTrie,
	enableCustomBlockHTR,
	disableInitSyncBatchSaveBlocks,
	enableStateRefCopy,
	waitForSyncedFlag,
}...)

// E2EBeaconChainFlags contains a list of the beacon chain feature flags to be tested in E2E.
var E2EBeaconChainFlags = []string{
	"--cache-filtered-block-tree",
	"--enable-eth1-data-vote-cache",
	"--enable-byte-mempool",
	"--enable-state-gen-sig-verify",
	"--check-head-state",
	"--enable-state-field-trie",
	"--enable-state-ref-copy",
	"--enable-new-state-mgmt",
	"--enable-custom-block-htr",
}

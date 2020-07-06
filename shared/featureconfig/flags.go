package featureconfig

import (
	"github.com/urfave/cli/v2"
)

var (
	devModeFlag = &cli.BoolFlag{
		Name:  "dev",
		Usage: "Enable experimental features still in development. These features may not be stable.",
	}
	disableBroadcastSlashingFlag = &cli.BoolFlag{
		Name:  "disable-broadcast-slashings",
		Usage: "Disables broadcasting slashings submitted to the beacon node.",
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
	disableInitSyncVerifyEverythingFlag = &cli.BoolFlag{
		Name: "disable-initial-sync-verify-all-signatures",
		Usage: "Initial sync to finalized checkpoint with verifying block's signature, RANDAO " +
			"and attestation's aggregated signatures. With this flag, only the proposer " +
			"signature is verified until the node reaches the end of the finalized chain.",
	}
	enableSlasherFlag = &cli.BoolFlag{
		Name: "enable-slasher",
		Usage: "Enables connection to a slasher service in order to retrieve slashable events. Slasher is connected to the beacon node using gRPC and " +
			"the slasher-provider flag can be used to pass its address.",
	}
	cacheFilteredBlockTreeFlag = &cli.BoolFlag{
		Name: "cache-filtered-block-tree",
		Usage: "Cache filtered block tree by maintaining it rather than continually recalculating on the fly, " +
			"this is used for fork choice.",
	}

	enableLocalProtectionFlag = &cli.BoolFlag{
		Name: "enable-local-protection",
		Usage: "Enables functionality to prevent the validator client from signing and " +
			"broadcasting any messages that could be considered slashable according to its own history.",
	}
	enableExternalSlasherProtectionFlag = &cli.BoolFlag{
		Name: "enable-external-slasher-protection",
		Usage: "Enables the validator to connect to external slasher to prevent it from " +
			"transmitting a slashable offence over the network.",
	}
	disableStrictAttestationPubsubVerificationFlag = &cli.BoolFlag{
		Name:  "disable-strict-attestation-pubsub-verification",
		Usage: "Disable strict signature verification of attestations in pubsub. See PR 4782 for details.",
	}
	disableUpdateHeadPerAttestation = &cli.BoolFlag{
		Name:  "disable-update-head-attestation",
		Usage: "Disable update fork choice head on per attestation. See PR 4802 for details.",
	}
	disableDomainDataCacheFlag = &cli.BoolFlag{
		Name: "disable-domain-data-cache",
		Usage: "Disable caching of domain data requests per epoch. This feature reduces the total " +
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
	disableNoiseHandshake = &cli.BoolFlag{
		Name: "disable-noise",
		Usage: "This disables the beacon node from using NOISE and instead uses SECIO instead for performing handshakes between peers and " +
			"securing transports between peers",
	}
	dontPruneStateStartUp = &cli.BoolFlag{
		Name:  "dont-prune-state-start-up",
		Usage: "Don't prune historical states upon start up",
	}
	disableNewStateMgmt = &cli.BoolFlag{
		Name:  "disable-new-state-mgmt",
		Usage: "This disables the usage of state mgmt service across Prysm",
	}
	waitForSyncedFlag = &cli.BoolFlag{
		Name:  "wait-for-synced",
		Usage: "Uses WaitForSynced for validator startup, to ensure a validator is able to communicate with the beacon node as quick as possible",
	}
	enableHistoricalDetectionFlag = &cli.BoolFlag{
		Name:  "enable-historical-detection",
		Usage: "Enables historical attestation detection for the slasher",
	}
	disableLookbackFlag = &cli.BoolFlag{
		Name:  "disable-lookback",
		Usage: "Disables use of the lookback feature and updates attestation history for validators from head to epoch 0",
	}
	skipRegenHistoricalStates = &cli.BoolFlag{
		Name:  "skip-regen-historical-states",
		Usage: "Skips regeneration and saving of historical states from genesis to last finalized. This enables a quick switch-over to using `--enable-new-state-mgmt`",
	}
	disableReduceAttesterStateCopy = &cli.BoolFlag{
		Name:  "disable-reduce-attester-state-copy",
		Usage: "Disables the feature to reduce the amount of state copies for attester rpc",
	}
	disableGRPCConnectionLogging = &cli.BoolFlag{
		Name:  "disable-grpc-connection-logging",
		Usage: "Disables displaying logs for newly connected grpc clients",
	}
	attestationAggregationStrategy = &cli.StringFlag{
		Name:  "attestation-aggregation-strategy",
		Usage: "Which strategy to use when aggregating attestations, one of: naive, max_cover.",
		Value: "naive",
	}
	newBeaconStateLocks = &cli.BoolFlag{
		Name:  "new-beacon-state-locks",
		Usage: "Enable new beacon state locking",
	}
	forceMaxCoverAttestationAggregation = &cli.BoolFlag{
		Name:  "attestation-aggregation-force-maxcover",
		Usage: "When enabled, forces --attestation-aggregation-strategy=max_cover setting.",
	}
	altonaTestnet = &cli.BoolFlag{
		Name:  "altona",
		Usage: "This defines the flag through which we can run on the Altona Multiclient Testnet",
	}
)

// devModeFlags holds list of flags that are set when development mode is on.
var devModeFlags = []cli.Flag{
	forceMaxCoverAttestationAggregation,
	newBeaconStateLocks,
}

// Deprecated flags list.
const deprecatedUsage = "DEPRECATED. DO NOT USE."

var (
	deprecatedP2PEncoding = &cli.StringFlag{
		Name:   "p2p-encoding",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedP2PPubsub = &cli.StringFlag{
		Name:   "p2p-pubsub",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableKadDht = &cli.BoolFlag{
		Name:   "enable-kad-dht",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedWeb3ProviderFlag = &cli.StringFlag{
		Name:   "web3provider",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
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
	deprecatedDisableProtectProposerFlag = &cli.BoolFlag{
		Name:   "disable-protect-proposer",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableProtectAttesterFlag = &cli.BoolFlag{
		Name:   "disable-protect-attester",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableCustomBlockHTR = &cli.BoolFlag{
		Name:   "enable-custom-block-htr",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableInitSyncQueueFlag = &cli.BoolFlag{
		Name:   "disable-init-sync-queue",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableEth1DataVoteCacheFlag = &cli.BoolFlag{
		Name:   "enable-eth1-data-vote-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedAccountMetricsFlag = &cli.BoolFlag{
		Name:   "enable-account-metrics",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableDomainDataCacheFlag = &cli.BoolFlag{
		Name:   "enable-domain-data-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableByteMempool = &cli.BoolFlag{
		Name:   "enable-byte-mempool",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedBroadcastSlashingFlag = &cli.BoolFlag{
		Name:   "broadcast-slashing",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableHistoricalDetectionFlag = &cli.BoolFlag{
		Name:   "disable-historical-detection",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecateEnableStateRefCopy = &cli.BoolFlag{
		Name:   "enable-state-ref-copy",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecateEnableFieldTrie = &cli.BoolFlag{
		Name:   "enable-state-field-trie",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecateEnableNewStateMgmt = &cli.BoolFlag{
		Name:   "enable-new-state-mgmt",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedP2PWhitelist = &cli.StringFlag{
		Name:   "p2p-whitelist",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedP2PBlacklist = &cli.StringFlag{
		Name:   "p2p-blacklist",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedSchlesiTestnetFlag = &cli.BoolFlag{
		Name:   "schlesi-testnet",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecateReduceAttesterStateCopies = &cli.BoolFlag{
		Name:   "reduce-attester-state-copy",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableStateRefCopy = &cli.BoolFlag{
		Name:   "disable-state-ref-copy",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableFieldTrie = &cli.BoolFlag{
		Name:   "disable-state-field-trie",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecateddisableInitSyncBatchSaveBlocks = &cli.BoolFlag{
		Name:   "disable-init-sync-batch-save-blocks",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableInitSyncWeightedRoundRobin = &cli.BoolFlag{
		Name:   "disable-init-sync-wrr",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableNoise = &cli.BoolFlag{
		Name:   "enable-noise",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedArchival = &cli.BoolFlag{
		Name:   "archive",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedArchiveValiatorSetChanges = &cli.BoolFlag{
		Name:   "archive-validator-set-changes",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedArchiveBlocks = &cli.BoolFlag{
		Name:   "archive-blocks",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedArchiveAttestation = &cli.BoolFlag{
		Name:   "archive-attestations",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableProtectProposerFlag = &cli.BoolFlag{
		Name:   "enable-protect-proposer",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableProtectAttesterFlag = &cli.BoolFlag{
		Name:   "enable-protect-attester",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedInitSyncVerifyEverythingFlag = &cli.BoolFlag{
		Name:   "initial-sync-verify-all-signatures",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

var deprecatedFlags = []cli.Flag{
	deprecatedP2PEncoding,
	deprecatedP2PPubsub,
	deprecatedEnableKadDht,
	deprecatedWeb3ProviderFlag,
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
	deprecatedDisableProtectProposerFlag,
	deprecatedDisableProtectAttesterFlag,
	deprecatedDisableInitSyncQueueFlag,
	deprecatedEnableCustomBlockHTR,
	deprecatedEnableEth1DataVoteCacheFlag,
	deprecatedAccountMetricsFlag,
	deprecatedEnableDomainDataCacheFlag,
	deprecatedEnableByteMempool,
	deprecatedBroadcastSlashingFlag,
	deprecatedDisableHistoricalDetectionFlag,
	deprecateEnableStateRefCopy,
	deprecateEnableFieldTrie,
	deprecateEnableNewStateMgmt,
	deprecatedP2PWhitelist,
	deprecatedP2PBlacklist,
	deprecatedSchlesiTestnetFlag,
	deprecateReduceAttesterStateCopies,
	deprecatedDisableInitSyncWeightedRoundRobin,
	deprecatedDisableStateRefCopy,
	deprecatedDisableFieldTrie,
	deprecateddisableInitSyncBatchSaveBlocks,
	deprecatedEnableNoise,
	deprecatedArchival,
	deprecatedArchiveBlocks,
	deprecatedArchiveValiatorSetChanges,
	deprecatedArchiveAttestation,
	deprecatedEnableProtectProposerFlag,
	deprecatedEnableProtectAttesterFlag,
	deprecatedInitSyncVerifyEverythingFlag,
}

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = append(deprecatedFlags, []cli.Flag{
	enableLocalProtectionFlag,
	enableExternalSlasherProtectionFlag,
	disableDomainDataCacheFlag,
	waitForSyncedFlag,
	altonaTestnet,
}...)

// SlasherFlags contains a list of all the feature flags that apply to the slasher client.
var SlasherFlags = append(deprecatedFlags, []cli.Flag{
	enableHistoricalDetectionFlag,
	disableLookbackFlag,
}...)

// E2EValidatorFlags contains a list of the validator feature flags to be tested in E2E.
var E2EValidatorFlags = []string{
	"--wait-for-synced",
	"--enable-local-protection",
}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = append(deprecatedFlags, []cli.Flag{
	devModeFlag,
	writeSSZStateTransitionsFlag,
	disableForkChoiceUnsafeFlag,
	disableDynamicCommitteeSubnets,
	disableSSZCache,
	disableInitSyncVerifyEverythingFlag,
	skipBLSVerifyFlag,
	kafkaBootstrapServersFlag,
	enableBackupWebhookFlag,
	enableSlasherFlag,
	cacheFilteredBlockTreeFlag,
	disableStrictAttestationPubsubVerificationFlag,
	disableUpdateHeadPerAttestation,
	enableStateGenSigVerify,
	checkHeadState,
	disableNoiseHandshake,
	dontPruneStateStartUp,
	disableBroadcastSlashingFlag,
	waitForSyncedFlag,
	skipRegenHistoricalStates,
	disableNewStateMgmt,
	disableReduceAttesterStateCopy,
	disableGRPCConnectionLogging,
	attestationAggregationStrategy,
	newBeaconStateLocks,
	forceMaxCoverAttestationAggregation,
	altonaTestnet,
}...)

// E2EBeaconChainFlags contains a list of the beacon chain feature flags to be tested in E2E.
var E2EBeaconChainFlags = []string{
	"--cache-filtered-block-tree",
	"--enable-state-gen-sig-verify",
	"--check-head-state",
	"--attestation-aggregation-strategy=max_cover",
	"--dev",
}

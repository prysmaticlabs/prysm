package featureconfig

import (
	"github.com/urfave/cli/v2"
)

var (
	// AltonaTestnet flag for the multiclient eth2 testnet configuration.
	AltonaTestnet = &cli.BoolFlag{
		Name:  "altona",
		Usage: "This defines the flag through which we can run on the Altona Multiclient Testnet",
	}
	// OnyxTestnet flag for the Prysmatic Labs single-client testnet configuration.
	OnyxTestnet = &cli.BoolFlag{
		Name:  "onyx",
		Usage: "This defines the flag through which we can run on the Onyx Prysm Testnet",
	}
	// MedallaTestnet flag for the multiclient eth2 testnet.
	MedallaTestnet = &cli.BoolFlag{
		Name:  "medalla",
		Usage: "This defines the flag through which we can run on the Medalla Multiclient Testnet",
	}
	// SpadinaTestnet flag for the multiclient eth2 testnet.
	SpadinaTestnet = &cli.BoolFlag{
		Name:  "spadina",
		Usage: "This defines the flag through which we can run on the Spadina Multiclient Testnet",
	}
	// ZinkenTestnet flag for the multiclient eth2 testnet.
	ZinkenTestnet = &cli.BoolFlag{
		Name:  "zinken",
		Usage: "This defines the flag through which we can run on the Zinken Multiclient Testnet",
	}
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
	enableLocalProtectionFlag = &cli.BoolFlag{
		Name: "enable-local-protection",
		Usage: "Enables functionality to prevent the validator client from signing and " +
			"broadcasting any messages that could be considered slashable according to its own history.",
		Value: true,
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
	disableNoiseHandshake = &cli.BoolFlag{
		Name: "disable-noise",
		Usage: "This disables the beacon node from using NOISE and instead uses SECIO instead for performing handshakes between peers and " +
			"securing transports between peers",
	}
	waitForSyncedFlag = &cli.BoolFlag{
		Name:  "wait-for-synced",
		Usage: "Uses WaitForSynced for validator startup, to ensure a validator is able to communicate with the beacon node as quick as possible",
	}
	disableLookbackFlag = &cli.BoolFlag{
		Name:  "disable-lookback",
		Usage: "Disables use of the lookback feature and updates attestation history for validators from head to epoch 0",
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
		Value: "max_cover",
	}
	disableNewBeaconStateLocks = &cli.BoolFlag{
		Name:  "disable-new-beacon-state-locks",
		Usage: "Disable new beacon state locking",
	}
	disableBatchBlockVerify = &cli.BoolFlag{
		Name:  "disable-batch-block-verify",
		Usage: "Disable full signature verification of blocks in batches instead of singularly.",
	}
	initSyncVerbose = &cli.BoolFlag{
		Name:  "init-sync-verbose",
		Usage: "Enable logging every processed block during initial syncing.",
	}
	enableBlst = &cli.BoolFlag{
		Name:  "blst",
		Usage: "Enable new BLS library, blst, from Supranational",
	}
	disableFinalizedDepositsCache = &cli.BoolFlag{
		Name:  "disable-finalized-deposits-cache",
		Usage: "Disables utilization of cached finalized deposits",
	}
	enableEth1DataMajorityVote = &cli.BoolFlag{
		Name:  "enable-eth1-data-majority-vote",
		Usage: "When enabled, voting on eth1 data will use the Voting With The Majority algorithm.",
	}
	disableAccountsV2 = &cli.BoolFlag{
		Name:  "disable-accounts-v2",
		Usage: "Disables usage of v2 for Prysm validator accounts",
	}
	enableAttBroadcastDiscoveryAttempts = &cli.BoolFlag{
		Name:  "enable-att-broadcast-discovery-attempts",
		Usage: "Enable experimental attestation subnet discovery before broadcasting.",
	}
	enablePeerScorer = &cli.BoolFlag{
		Name:  "enable-peer-scorer",
		Usage: "Enable experimental P2P peer scorer",
	}
	checkPtInfoCache = &cli.BoolFlag{
		Name:  "use-check-point-cache",
		Usage: "Enables check point info caching",
	}
	enablePruningDepositProofs = &cli.BoolFlag{
		Name:  "enable-pruning-deposit-proofs",
		Usage: "Enables pruning deposit proofs when they are no longer needed. This significantly reduces deposit size.",
	}
)

// devModeFlags holds list of flags that are set when development mode is on.
var devModeFlags = []cli.Flag{
	checkPtInfoCache,
	enableEth1DataMajorityVote,
	enableAttBroadcastDiscoveryAttempts,
	enablePeerScorer,
	enablePruningDepositProofs,
}

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = append(deprecatedFlags, []cli.Flag{
	enableLocalProtectionFlag,
	enableExternalSlasherProtectionFlag,
	disableDomainDataCacheFlag,
	waitForSyncedFlag,
	AltonaTestnet,
	OnyxTestnet,
	MedallaTestnet,
	SpadinaTestnet,
	ZinkenTestnet,
	disableAccountsV2,
	enableBlst,
}...)

// SlasherFlags contains a list of all the feature flags that apply to the slasher client.
var SlasherFlags = append(deprecatedFlags, []cli.Flag{
	disableLookbackFlag,
	AltonaTestnet,
	OnyxTestnet,
	MedallaTestnet,
	SpadinaTestnet,
	ZinkenTestnet,
}...)

// E2EValidatorFlags contains a list of the validator feature flags to be tested in E2E.
var E2EValidatorFlags = []string{
	"--wait-for-synced",
}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = append(deprecatedFlags, []cli.Flag{
	devModeFlag,
	writeSSZStateTransitionsFlag,
	disableDynamicCommitteeSubnets,
	disableSSZCache,
	skipBLSVerifyFlag,
	kafkaBootstrapServersFlag,
	enableBackupWebhookFlag,
	disableStrictAttestationPubsubVerificationFlag,
	disableUpdateHeadPerAttestation,
	enableStateGenSigVerify,
	disableNoiseHandshake,
	disableBroadcastSlashingFlag,
	waitForSyncedFlag,
	disableReduceAttesterStateCopy,
	disableGRPCConnectionLogging,
	attestationAggregationStrategy,
	disableNewBeaconStateLocks,
	AltonaTestnet,
	OnyxTestnet,
	MedallaTestnet,
	SpadinaTestnet,
	ZinkenTestnet,
	disableBatchBlockVerify,
	initSyncVerbose,
	disableFinalizedDepositsCache,
	enableBlst,
	enableEth1DataMajorityVote,
	enableAttBroadcastDiscoveryAttempts,
	enablePeerScorer,
	checkPtInfoCache,
	enablePruningDepositProofs,
}...)

// E2EBeaconChainFlags contains a list of the beacon chain feature flags to be tested in E2E.
var E2EBeaconChainFlags = []string{
	"--enable-state-gen-sig-verify",
	"--attestation-aggregation-strategy=max_cover",
	"--dev",
	"--enable-eth1-data-majority-vote",
	"--use-check-point-cache",
	"--enable-pruning-deposit-proofs",
}

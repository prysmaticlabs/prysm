package features

import (
	"time"

	"github.com/urfave/cli/v2"
)

var (
	// PraterTestnet flag for the multiclient Ethereum consensus testnet.
	PraterTestnet = &cli.BoolFlag{
		Name:    "prater",
		Usage:   "Run Prysm configured for the Prater / Goerli test network",
		Aliases: []string{"goerli"},
	}
	// RopstenTestnet flag for the multiclient Ethereum consensus testnet.
	RopstenTestnet = &cli.BoolFlag{
		Name:  "ropsten",
		Usage: "Run Prysm configured for the Ropsten beacon chain test network",
	}
	// SepoliaTestnet flag for the multiclient Ethereum consensus testnet.
	SepoliaTestnet = &cli.BoolFlag{
		Name:  "sepolia",
		Usage: "Run Prysm configured for the Sepolia beacon chain test network",
	}
	// Mainnet flag for easier tooling, no-op
	Mainnet = &cli.BoolFlag{
		Value: true,
		Name:  "mainnet",
		Usage: "Run on Ethereum Beacon Chain Main Net. This is the default and can be omitted.",
	}
	devModeFlag = &cli.BoolFlag{
		Name:  "dev",
		Usage: "Enable experimental features still in development. These features may not be stable.",
	}
	writeSSZStateTransitionsFlag = &cli.BoolFlag{
		Name:  "interop-write-ssz-state-transitions",
		Usage: "Write ssz states to disk after attempted state transition",
	}
	enableExternalSlasherProtectionFlag = &cli.BoolFlag{
		Name: "enable-external-slasher-protection",
		Usage: "Enables the validator to connect to a beacon node using the --slasher flag" +
			"for remote slashing protection",
	}
	disableGRPCConnectionLogging = &cli.BoolFlag{
		Name:  "disable-grpc-connection-logging",
		Usage: "Disables displaying logs for newly connected grpc clients",
	}
	disablePeerScorer = &cli.BoolFlag{
		Name:  "disable-peer-scorer",
		Usage: "Disables experimental P2P peer scorer",
	}
	writeWalletPasswordOnWebOnboarding = &cli.BoolFlag{
		Name: "write-wallet-password-on-web-onboarding",
		Usage: "(Danger): Writes the wallet password to the wallet directory on completing Prysm web onboarding. " +
			"We recommend against this flag unless you are an advanced user.",
	}
	dynamicKeyReloadDebounceInterval = &cli.DurationFlag{
		Name: "dynamic-key-reload-debounce-interval",
		Usage: "(Advanced): Specifies the time duration the validator waits to reload new keys if they have " +
			"changed on disk. Default 1s, can be any type of duration such as 1.5s, 1000ms, 1m.",
		Value: time.Second,
	}
	disableBroadcastSlashingFlag = &cli.BoolFlag{
		Name:  "disable-broadcast-slashings",
		Usage: "Disables broadcasting slashings submitted to the beacon node.",
	}
	attestTimely = &cli.BoolFlag{
		Name:  "attest-timely",
		Usage: "Fixes validator can attest timely after current block processes. See #8185 for more details",
	}
	enableSlasherFlag = &cli.BoolFlag{
		Name:  "slasher",
		Usage: "Enables a slasher in the beacon node for detecting slashable offenses",
	}
	enableSlashingProtectionPruning = &cli.BoolFlag{
		Name:  "enable-slashing-protection-history-pruning",
		Usage: "Enables the pruning of the validator client's slashing protection database",
	}
	enableDoppelGangerProtection = &cli.BoolFlag{
		Name: "enable-doppelganger",
		Usage: "Enables the validator to perform a doppelganger check. (Warning): This is not " +
			"a foolproof method to find duplicate instances in the network. Your validator will still be" +
			" vulnerable if it is being run in unsafe configurations.",
	}
	disableStakinContractCheck = &cli.BoolFlag{
		Name:  "disable-staking-contract-check",
		Usage: "Disables checking of staking contract deposits when proposing blocks, useful for devnets",
	}
	enableHistoricalSpaceRepresentation = &cli.BoolFlag{
		Name: "enable-historical-state-representation",
		Usage: "Enables the beacon chain to save historical states in a space efficient manner." +
			" (Warning): Once enabled, this feature migrates your database in to a new schema and " +
			"there is no going back. At worst, your entire database might get corrupted.",
	}
	disablePullTips = &cli.BoolFlag{
		Name:  "experimental-enable-boundary-checks",
		Usage: "Experimental enable of boundary checks, useful for debugging, may cause bad votes.",
	}
	disableDefensivePull = &cli.BoolFlag{
		Name:   "disable-back-pull",
		Usage:  "Experimental disable of past boundary checks, useful for debugging, may cause bad votes.",
		Hidden: true,
	}
	disableVecHTR = &cli.BoolFlag{
		Name:  "disable-vectorized-htr",
		Usage: "Disables the new go sha256 library which utilizes optimized routines for merkle trees",
	}
	disableForkChoiceDoublyLinkedTree = &cli.BoolFlag{
		Name:  "disable-forkchoice-doubly-linked-tree",
		Usage: "Disables the new forkchoice store structure that uses doubly linked trees",
	}
	disableGossipBatchAggregation = &cli.BoolFlag{
		Name:  "disable-gossip-batch-aggregation",
		Usage: "Disables new methods to further aggregate our gossip batches before verifying them.",
	}
	EnableOnlyBlindedBeaconBlocks = &cli.BoolFlag{
		Name:  "enable-only-blinded-beacon-blocks",
		Usage: "Enables storing only blinded beacon blocks in the database without full execution layer transactions",
	}
	enableStartupOptimistic = &cli.BoolFlag{
		Name:   "startup-optimistic",
		Usage:  "Treats every block as optimistically synced at launch. Use with caution",
		Value:  false,
		Hidden: true,
	}
	enableFullSSZDataLogging = &cli.BoolFlag{
		Name:  "enable-full-ssz-data-logging",
		Usage: "Enables displaying logs for full ssz data on rejected gossip messages",
	}
)

// devModeFlags holds list of flags that are set when development mode is on.
var devModeFlags = []cli.Flag{}

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = append(deprecatedFlags, []cli.Flag{
	writeWalletPasswordOnWebOnboarding,
	enableExternalSlasherProtectionFlag,
	PraterTestnet,
	RopstenTestnet,
	SepoliaTestnet,
	Mainnet,
	dynamicKeyReloadDebounceInterval,
	attestTimely,
	enableSlashingProtectionPruning,
	enableDoppelGangerProtection,
}...)

// E2EValidatorFlags contains a list of the validator feature flags to be tested in E2E.
var E2EValidatorFlags = []string{
	"--enable-doppelganger",
}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = append(deprecatedBeaconFlags, append(deprecatedFlags, []cli.Flag{
	devModeFlag,
	writeSSZStateTransitionsFlag,
	disableGRPCConnectionLogging,
	PraterTestnet,
	RopstenTestnet,
	SepoliaTestnet,
	Mainnet,
	disablePeerScorer,
	disableBroadcastSlashingFlag,
	enableSlasherFlag,
	enableHistoricalSpaceRepresentation,
	disablePullTips,
	disableVecHTR,
	disableForkChoiceDoublyLinkedTree,
	disableGossipBatchAggregation,
	EnableOnlyBlindedBeaconBlocks,
	enableStartupOptimistic,
	disableDefensivePull,
	enableFullSSZDataLogging,
}...)...)

// E2EBeaconChainFlags contains a list of the beacon chain feature flags to be tested in E2E.
var E2EBeaconChainFlags = []string{
	"--dev",
}

package features

import "github.com/urfave/cli/v2"

// Deprecated flags list.
const deprecatedUsage = "DEPRECATED. DO NOT USE."

var (
	// To deprecate a feature flag, first copy the example below, then insert deprecated flag in `deprecatedFlags`.
	exampleDeprecatedFeatureFlag = &cli.StringFlag{
		Name:   "name",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedBackupWebHookFlag = &cli.BoolFlag{
		Name:   "enable-db-backup-webhook",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedBoltMmapFlag = &cli.StringFlag{
		Name:   "bolt-mmap-initial-size",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableDiscV5Flag = &cli.BoolFlag{
		Name:   "disable-discv5",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableAttHistoryCacheFlag = &cli.BoolFlag{
		Name:   "disable-attesting-history-db-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableVectorizedHtr = &cli.BoolFlag{
		Name:   "enable-vectorized-htr",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnablePeerScorer = &cli.BoolFlag{
		Name:   "enable-peer-scorer",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableForkchoiceDoublyLinkedTree = &cli.BoolFlag{
		Name:   "enable-forkchoice-doubly-linked-tree",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableDefensivePull = &cli.BoolFlag{
		Name:   "enable-back-pull",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDutyCountdown = &cli.BoolFlag{
		Name:   "enable-duty-count-down",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedHeadSync = &cli.BoolFlag{
		Name:   "head-sync",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedGossipBatchAggregation = &cli.BoolFlag{
		Name:   "enable-gossip-batch-aggregation",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableLargerGossipHistory = &cli.BoolFlag{
		Name:   "enable-larger-gossip-history",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedFallbackProvider = &cli.StringFlag{
		Name:   "fallback-web3provider",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableNativeState = &cli.StringFlag{
		Name:   "disable-native-state",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

// Deprecated flags for both the beacon node and validator client.
var deprecatedFlags = []cli.Flag{
	exampleDeprecatedFeatureFlag,
	deprecatedBoltMmapFlag,
	deprecatedDisableDiscV5Flag,
	deprecatedDisableAttHistoryCacheFlag,
	deprecatedEnableVectorizedHtr,
	deprecatedEnablePeerScorer,
	deprecatedEnableForkchoiceDoublyLinkedTree,
	deprecatedDutyCountdown,
	deprecatedHeadSync,
	deprecatedGossipBatchAggregation,
	deprecatedEnableLargerGossipHistory,
	deprecatedFallbackProvider,
	deprecatedEnableDefensivePull,
	deprecatedDisableNativeState,
}

// deprecatedBeaconFlags contains flags that are still used by other components
// and therefore cannot be added to deprecatedFlags
var deprecatedBeaconFlags = []cli.Flag{
	deprecatedBackupWebHookFlag,
}

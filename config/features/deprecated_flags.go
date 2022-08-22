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
	deprecatedBackupWebHookFlag = &cli.StringFlag{
		Name:   "enable-db-backup-webhook",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedBoltMmapFlag = &cli.StringFlag{
		Name:   "bolt-mmap-initial-size",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableDiscV5Flag = &cli.StringFlag{
		Name:   "disable-discv5",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableAttHistoryCacheFlag = &cli.StringFlag{
		Name:   "disable-attesting-history-db-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableVectorizedHtr = &cli.StringFlag{
		Name:   "enable-vectorized-htr",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnablePeerScorer = &cli.StringFlag{
		Name:   "enable-peer-scorer",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableGossipBatchAggregation = &cli.StringFlag{
		Name:   "enable-gossip-batch-aggregation",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableForkchoiceDoublyLinkedTree = &cli.StringFlag{
		Name:   "enable-forkchoice-doubly-linked-tree",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDutyCountdown = &cli.StringFlag{
		Name:   "enable-duty-count-down",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedHeadSync = &cli.StringFlag{
		Name:   "head-sync",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedGossipBatchAggregation = &cli.StringFlag{
		Name:   "enable-gossip-batch-aggregation",
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
	deprecatedEnableGossipBatchAggregation,
	deprecatedEnableForkchoiceDoublyLinkedTree,
	deprecatedDutyCountdown,
	deprecatedHeadSync,
	deprecatedGossipBatchAggregation,
}

var deprecatedBeaconFlags = []cli.Flag{
	deprecatedBackupWebHookFlag,
}

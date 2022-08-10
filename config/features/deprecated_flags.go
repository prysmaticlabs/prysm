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

	deprecatedEnableActiveBalanceCache = &cli.BoolFlag{
		Name:   "enable-active-balance-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}

	deprecatedCorrectlyPruneCanonicalAtts = &cli.BoolFlag{
		Name:   "correctly-prune-canonical-atts",
		Usage:  deprecatedUsage,
		Hidden: true,
	}

	deprecatedCorrectlyInsertOrphanedAtts = &cli.BoolFlag{
		Name:   "correctly-insert-orphaned-atts",
		Usage:  deprecatedUsage,
		Hidden: true,
	}

	deprecatedNextSlotStateCache = &cli.BoolFlag{
		Name:   "enable-next-slot-state-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}

	deprecatedEnableBatchGossipVerification = &cli.BoolFlag{
		Name:   "enable-batch-gossip-verification",
		Usage:  deprecatedUsage,
		Hidden: true,
	}

	deprecatedEnableGetBlockOptimizations = &cli.BoolFlag{
		Name:   "enable-get-block-optimizations",
		Usage:  deprecatedUsage,
		Hidden: true,
	}

	deprecatedEnableBalanceTrieComputation = &cli.BoolFlag{
		Name:   "enable-balance-trie-computation",
		Usage:  deprecatedUsage,
		Hidden: true,
	}

	deprecatedDisableNextSlotStateCache = &cli.BoolFlag{
		Name:   "disable-next-slot-state-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedAttestationAggregationStrategy = &cli.BoolFlag{
		Name:   "attestation-aggregation-strategy",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedForceOptMaxCoverAggregationStategy = &cli.BoolFlag{
		Name:   "attestation-aggregation-force-opt-maxcover",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedPyrmontTestnet = &cli.BoolFlag{
		Name:   "pyrmont",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableGetBlockOptimizations = &cli.BoolFlag{
		Name:   "disable-get-block-optimizations",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableProposerAttsSelectionUsingMaxCover = &cli.BoolFlag{
		Name:   "disable-proposer-atts-selection-using-max-cover",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableOptimizedBalanceUpdate = &cli.BoolFlag{
		Name:   "disable-optimized-balance-update",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableActiveBalanceCache = &cli.BoolFlag{
		Name:   "disable-active-balance-cache",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableBalanceTrieComputation = &cli.BoolFlag{
		Name:   "disable-balance-trie-computation",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableBatchGossipVerification = &cli.BoolFlag{
		Name:   "disable-batch-gossip-verification",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableCorrectlyPruneCanonicalAtts = &cli.BoolFlag{
		Name:   "disable-correctly-prune-canonical-atts",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableCorrectlyInsertOrphanedAtts = &cli.BoolFlag{
		Name:   "disable-correctly-insert-orphaned-atts",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableNativeState = &cli.BoolFlag{
		Name:   "enable-native-state",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnablePeerScorer = &cli.BoolFlag{
		Name:   "enable-peer-scorer",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableGossipBatchAggregation = &cli.BoolFlag{
		Name:   "enable-gossip-batch-aggregation",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnablePullTips = &cli.BoolFlag{
		Name:   "experimental-disable-boundary-checks",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

var deprecatedFlags = []cli.Flag{
	exampleDeprecatedFeatureFlag,
	deprecatedEnableActiveBalanceCache,
	deprecatedCorrectlyPruneCanonicalAtts,
	deprecatedCorrectlyInsertOrphanedAtts,
	deprecatedNextSlotStateCache,
	deprecatedEnableBatchGossipVerification,
	deprecatedEnableGetBlockOptimizations,
	deprecatedEnableBalanceTrieComputation,
	deprecatedDisableNextSlotStateCache,
	deprecatedAttestationAggregationStrategy,
	deprecatedForceOptMaxCoverAggregationStategy,
	deprecatedPyrmontTestnet,
	deprecatedDisableProposerAttsSelectionUsingMaxCover,
	deprecatedDisableGetBlockOptimizations,
	deprecatedDisableOptimizedBalanceUpdate,
	deprecatedDisableActiveBalanceCache,
	deprecatedDisableBalanceTrieComputation,
	deprecatedDisableBatchGossipVerification,
	deprecatedDisableCorrectlyInsertOrphanedAtts,
	deprecatedDisableCorrectlyPruneCanonicalAtts,
	deprecatedEnableNativeState,
	deprecatedEnablePeerScorer,
	deprecatedEnableGossipBatchAggregation,
	deprecatedEnablePullTips,
}

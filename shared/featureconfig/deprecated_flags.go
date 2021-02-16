package featureconfig

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
	deprecatedEnableSyncBacktracking = &cli.StringFlag{
		Name:   "enable-sync-backtracking",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableSyncBacktracking = &cli.StringFlag{
		Name:   "disable-sync-backtracking",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisablePruningDepositProofs = &cli.BoolFlag{
		Name:   "disable-pruning-deposit-proofs",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableEth1DataMajorityVote = &cli.BoolFlag{
		Name:   "disable-eth1-data-majority-vote",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

var deprecatedFlags = []cli.Flag{
	exampleDeprecatedFeatureFlag,
	deprecatedEnableSyncBacktracking,
	deprecatedDisableSyncBacktracking,
	deprecatedDisablePruningDepositProofs,
	deprecatedDisableEth1DataMajorityVote,
}

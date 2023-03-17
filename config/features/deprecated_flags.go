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
	deprecatedDisablePeerScorer = &cli.BoolFlag{
		Name:   "disable-peer-scorer",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableGossipBatchAggregation = &cli.BoolFlag{
		Name:   "disable-gossip-batch-aggregation",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

// Deprecated flags for both the beacon node and validator client.
var deprecatedFlags = []cli.Flag{
	exampleDeprecatedFeatureFlag,
	deprecatedDisablePeerScorer,
	deprecatedDisableGossipBatchAggregation,
}

// deprecatedBeaconFlags contains flags that are still used by other components
// and therefore cannot be added to deprecatedFlags
var deprecatedBeaconFlags = []cli.Flag{}

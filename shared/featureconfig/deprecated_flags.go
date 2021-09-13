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
)

var deprecatedFlags = []cli.Flag{
	exampleDeprecatedFeatureFlag,
	deprecatedEnableActiveBalanceCache,
	deprecatedCorrectlyPruneCanonicalAtts,
}

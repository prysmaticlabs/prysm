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
)

var deprecatedFlags = []cli.Flag{
	exampleDeprecatedFeatureFlag,
	deprecatedEnableNativeState,
	deprecatedEnablePeerScorer,
}

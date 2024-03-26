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
	deprecatedEnableOptionalEngineMethods = &cli.BoolFlag{
		Name:   "enable-optional-engine-methods",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableBuildBlockParallel = &cli.BoolFlag{
		Name:   "disable-build-block-parallel",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableReorgLateBlocks = &cli.BoolFlag{
		Name:   "disable-reorg-late-blocks",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableOptionalEngineMethods = &cli.BoolFlag{
		Name:   "disable-optional-engine-methods",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedDisableAggregateParallel = &cli.BoolFlag{
		Name:   "disable-aggregate-parallel",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedEnableEIP4881 = &cli.BoolFlag{
		Name:   "enable-eip-4881",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
	deprecatedVerboseSigVerification = &cli.BoolFlag{
		Name:   "enable-verbose-sig-verification",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

// Deprecated flags for both the beacon node and validator client.
var deprecatedFlags = []cli.Flag{
	exampleDeprecatedFeatureFlag,
	deprecatedEnableOptionalEngineMethods,
	deprecatedDisableBuildBlockParallel,
	deprecatedDisableReorgLateBlocks,
	deprecatedDisableOptionalEngineMethods,
	deprecatedDisableAggregateParallel,
	deprecatedEnableEIP4881,
	deprecatedVerboseSigVerification,
}

// deprecatedBeaconFlags contains flags that are still used by other components
// and therefore cannot be added to deprecatedFlags
var deprecatedBeaconFlags []cli.Flag

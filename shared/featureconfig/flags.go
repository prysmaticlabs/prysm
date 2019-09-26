package featureconfig

import (
	"github.com/urfave/cli"
)

var (
	// NoGenesisDelayFlag disables the standard genesis delay.
	NoGenesisDelayFlag = cli.BoolFlag{
		Name:  "no-genesis-delay",
		Usage: "Process genesis event 30s after the ETH1 block time, rather than wait to midnight of the next day.",
	}
	// DemoConfigFlag enables the demo configuration.
	DemoConfigFlag = cli.BoolFlag{
		Name:  "demo-config",
		Usage: "Use demo config with lower deposit thresholds.",
	}
	writeSSZStateTransitionsFlag = cli.BoolFlag{
		Name:  "interop-write-ssz-state-transitions",
		Usage: "Write ssz states to disk after attempted state transition",
	}
	// EnableAttestationCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableAttestationCacheFlag = cli.BoolFlag{
		Name:  "enable-attestation-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableEth1DataVoteCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableEth1DataVoteCacheFlag = cli.BoolFlag{
		Name:  "enable-eth1-data-vote-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
)

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = []cli.Flag{
	DemoConfigFlag,
}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = []cli.Flag{
	NoGenesisDelayFlag,
	DemoConfigFlag,
	writeSSZStateTransitionsFlag,
	EnableAttestationCacheFlag,
	EnableEth1DataVoteCacheFlag,
}

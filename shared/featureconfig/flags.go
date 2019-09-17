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
	// EnableActiveBalanceCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableActiveBalanceCacheFlag = cli.BoolFlag{
		Name:  "enable-active-balance-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableAttestationCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableAttestationCacheFlag = cli.BoolFlag{
		Name:  "enable-attestation-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableAncestorBlockCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableAncestorBlockCacheFlag = cli.BoolFlag{
		Name:  "enable-ancestor-block-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableEth1DataVoteCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableEth1DataVoteCacheFlag = cli.BoolFlag{
		Name:  "enable-eth1-data-vote-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableSeedCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableSeedCacheFlag = cli.BoolFlag{
		Name:  "enable-seed-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableStartShardCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableStartShardCacheFlag = cli.BoolFlag{
		Name:  "enable-start-shard-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableTotalBalanceCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableTotalBalanceCacheFlag = cli.BoolFlag{
		Name:  "enable-total-balance-cache",
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
	EnableActiveBalanceCacheFlag,
	EnableAttestationCacheFlag,
	EnableAncestorBlockCacheFlag,
	EnableEth1DataVoteCacheFlag,
	EnableSeedCacheFlag,
	EnableStartShardCacheFlag,
	EnableTotalBalanceCacheFlag,
}

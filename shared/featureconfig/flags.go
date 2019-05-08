package featureconfig

import (
	"github.com/urfave/cli"
)

var (
	// CacheTreeHashFlag determines whether to cache tree hashes for ssz.
	CacheTreeHashFlag = cli.BoolFlag{
		Name:  "enable-cache-tree-hash",
		Usage: "Cache tree hashes for ssz",
	}
	// VerifyAttestationSigsFlag determines whether to verify signatures for attestations.
	VerifyAttestationSigsFlag = cli.BoolFlag{
		Name:  "enable-attestation-signature-verification",
		Usage: "Verify signatures for attestations.",
	}
	// EnableComputeStateRootFlag enables the implemenation for the proposer RPC
	// method to compute the state root of a given block.
	// This feature is not
	// necessary for the first iteration of the test network, but critical to
	// future work. This flag can be removed once we are satisified that it works
	// well without issue.
	EnableComputeStateRootFlag = cli.BoolFlag{
		Name:  "enable-compute-state-root",
		Usage: "Enable server side compute state root. Default is a no-op implementation.",
	}
	// EnableCrosslinksFlag enables the processing of crosslinks in epoch processing. It is disabled by default.
	EnableCrosslinksFlag = cli.BoolFlag{
		Name:  "enable-crosslinks",
		Usage: "Enable crosslinks in epoch processing, default is disabled.",
	}
	// EnableCheckBlockStateRootFlag check block state root in block processing. It is disabled by default.
	EnableCheckBlockStateRootFlag = cli.BoolFlag{
		Name:  "enable-check-block-state-root",
		Usage: "Enable check block state root in block processing, default is disabled.",
	}
	// DisableHistoricalStatePruningFlag allows the database to keep old historical states.
	DisableHistoricalStatePruningFlag = cli.BoolFlag{
		Name:  "disable-historical-state-pruning",
		Usage: "Disable database pruning of historical states after finalized epochs.",
	}
	// DisableGossipSubFlag uses floodsub in place of gossipsub.
	DisableGossipSubFlag = cli.BoolFlag{
		Name:  "disable-gossip-sub",
		Usage: "Disable gossip sub messaging and use floodsub messaging",
	}
)

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = []cli.Flag{
	CacheTreeHashFlag,
}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = []cli.Flag{
	EnableComputeStateRootFlag,
	EnableCrosslinksFlag,
	EnableCheckBlockStateRootFlag,
	DisableHistoricalStatePruningFlag,
	DisableGossipSubFlag,
	CacheTreeHashFlag,
}

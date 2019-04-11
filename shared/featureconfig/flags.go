package featureconfig

import (
	"github.com/urfave/cli"
)

var (
	// VerifyAttestationSigsFlag determines whether to verify signatures for attestations.
	VerifyAttestationSigsFlag = cli.BoolFlag{
		Name:  "enable-attestation-signature-verification",
		Usage: "Verify signatures for attestations.",
	}
	// EnableComputeStateRootFlag enables the implemenation for the proposer RPC
	// method to compute the state root of a given block. This feature is not
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
	// EnableCommitteesCacheFlag enables crosslink committees cache for rpc server. It is disabled by default.
	EnableCommitteesCacheFlag = cli.BoolFlag{
		Name:  "enable-committees-cache",
		Usage: "Enable crosslink committees cache for rpc server, default is disabled.",
	}
	// EnableCheckBlockStateRootFlag check block state root in block processing. It is disabled by default.
	EnableCheckBlockStateRootFlag = cli.BoolFlag{
		Name:  "enable-check-block-state-root",
		Usage: "Enable check block state root in block processing, default is disabled.",
	}
	// EnableHistoricalStatePruningFlag allows the database to prune old historical states.
	EnableHistoricalStatePruningFlag = cli.BoolFlag{
		Name:  "enable-historical-state-pruning",
		Usage: "Enable database pruning of historical states after finalized epochs",
	}
)

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = []cli.Flag{}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = []cli.Flag{
	EnableComputeStateRootFlag,
	EnableCrosslinksFlag,
	EnableCommitteesCacheFlag,
	EnableCheckBlockStateRootFlag,
	EnableHistoricalStatePruningFlag,
}

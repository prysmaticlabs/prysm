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
	EnableComputeStateRootFlag = cli.BoolFlag{
		Name:  "enable-compute-state-root",
		Usage: "Enable server side compute state root. Default is a no-op implementation.",
	}
)

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = []cli.Flag{}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = []cli.Flag{
	EnableComputeStateRootFlag,
}

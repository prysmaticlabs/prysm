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
	// VerifyBlockSigsFlag determines whether to verify signatures for blocks.
	VerifyBlockSigsFlag = cli.BoolFlag{
		Name:  "enable-block-signature-verification",
		Usage: "Verify signatures for blocks.",
	}
)

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = []cli.Flag{
	VerifyAttestationSigsFlag,
	VerifyBlockSigsFlag,
}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = []cli.Flag{
	VerifyAttestationSigsFlag,
	VerifyBlockSigsFlag,
}

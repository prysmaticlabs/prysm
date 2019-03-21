/*
Package featureconfig defines which features are enabled for runtime
in order to selctively enable certain features to maintain a stable runtime.

The process for implementing new features using this package is as follows:
	1. Add a new CMD flag in flags.go, and place it in the proper list(s) var for its client.
	2. Add a condition for the flag in the proper Configure function(s) below.
	3. Place any "new" behavior in the `if flagEnabled` statement.
	4. Place any "previous" behavior in the `else` statement.
	5. Ensure any tests using the new feature fail if the flag isn't enabled.
	5a. Use the following to enable your flag for tests:
	cfg := &featureconfig.FeatureFlagConfig{
		VerifyAttestationSigs: true,
	}
	featureconfig.InitFeatureConfig(cfg)
*/
package featureconfig

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "flags")

// FeatureFlagConfig is a struct to represent what features the client will perform on runtime.
type FeatureFlagConfig struct {
	VerifyAttestationSigs bool // VerifyAttestationSigs declares if the client will verify attestations.
}

var featureConfig *FeatureFlagConfig

// FeatureConfig retrieves feature config.
func FeatureConfig() *FeatureFlagConfig {
	return featureConfig
}

// InitFeatureConfig sets the global config equal to the config that is passed in.
func InitFeatureConfig(c *FeatureFlagConfig) {
	featureConfig = c
}

// ConfigureBeaconFeatures sets the global config based
// on what flags are enabled for the beacon-chain client.
func ConfigureBeaconFeatures(ctx *cli.Context) {
	cfg := &FeatureFlagConfig{}
	if ctx.GlobalBool(VerifyAttestationSigsFlag.Name) {
		log.Info("Verifying signatures for attestations")
		cfg.VerifyAttestationSigs = true
	}

	InitFeatureConfig(cfg)
}

// ConfigureValidatorFeatures sets the global config based
// on what flags are enabled for the validator client.
func ConfigureValidatorFeatures(ctx *cli.Context) {
	cfg := &FeatureFlagConfig{}
	if ctx.GlobalBool(VerifyAttestationSigsFlag.Name) {
		log.Info("Verifying signatures for attestations")
		cfg.VerifyAttestationSigs = true
	}

	InitFeatureConfig(cfg)
}

/*
Package featureflags defines which features are enabled for runtime
in order to selctively enable certain features to maintain a stable runtime.

The process for implementing new features using this package is as follows:
	1. Create a new CMD flag in validator/types/flags.go and beacon-chain/utils/flags.go.
	2. If the feature has changes in beacon-chain, place the flag in beacon-chain/main.go
	and beacon-chain/usage.go in the "features" flag group.
	2a. Add a case for the flag in ConfigureBeaconFeatures.
	3. If the feature has changes in validator, place the flag in validator/main.go
	and validator/usage.go in the "features" flag group.
	3a. Add a case for the flag in the ConfigureValidatorFeatures function in this file.
	4. Place any "new" behavior in the `if flagEnabled` statement.
	5. Place any "previous" behavior in the `else` statement.
	6. Ensure any tests using the new feature fail if the flag isn't enabled.
*/
package featureflags

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/validator/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log = logrus.WithField("prefix", "node")

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
	if ctx.GlobalBool(utils.VerifyAttestationSigsFlag.Name) {
		log.Info("Verifying signatures for attestations")
		cfg.VerifyAttestationSigs = true
	}

	InitFeatureConfig(cfg)
}

// ConfigureValidatorFeatures sets the global config based
// on what flags are enabled for the validator client.
func ConfigureValidatorFeatures(ctx *cli.Context) {
	cfg := &FeatureFlagConfig{}
	if ctx.GlobalBool(types.VerifyAttestationSigsFlag.Name) {
		log.Info("Verifying signatures for attestations")
		cfg.VerifyAttestationSigs = true
	}

	InitFeatureConfig(cfg)
}

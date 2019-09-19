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
	NoGenesisDelay bool // NoGenesisDelay when processing a chain start genesis event.
	DemoConfig     bool // DemoConfig with lower deposit thresholds.

	// Cache toggles.
	EnableActiveBalanceCache bool // EnableActiveBalanceCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableAttestationCache   bool // EnableAttestationCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableAncestorBlockCache bool // EnableAncestorBlockCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableEth1DataVoteCache  bool // EnableEth1DataVoteCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableSeedCache          bool // EnableSeedCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableStartShardCache    bool // EnableStartShardCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableTotalBalanceCache  bool // EnableTotalBalanceCache; see https://github.com/prysmaticlabs/prysm/issues/3106.
}

var featureConfig *FeatureFlagConfig

// FeatureConfig retrieves feature config.
func FeatureConfig() *FeatureFlagConfig {
	if featureConfig == nil {
		return &FeatureFlagConfig{}
	}
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
	if ctx.GlobalBool(DemoConfigFlag.Name) {
		log.Warn("Using demo config")
		cfg.DemoConfig = true
	}
	if ctx.GlobalBool(NoGenesisDelayFlag.Name) {
		log.Warn("Using non standard genesis delay. This may cause problems in a multi-node environment.")
		cfg.NoGenesisDelay = true
	}
	if ctx.GlobalBool(EnableActiveBalanceCacheFlag.Name) {
		log.Warn("Enabled unsafe active balance cache")
		cfg.EnableActiveBalanceCache = true
	}
	if ctx.GlobalBool(EnableAttestationCacheFlag.Name) {
		log.Warn("Enabled unsafe attestation cache")
		cfg.EnableAttestationCache = true
	}
	if ctx.GlobalBool(EnableAncestorBlockCacheFlag.Name) {
		log.Warn("Enabled unsafe ancestor block cache")
		cfg.EnableAncestorBlockCache = true
	}
	if ctx.GlobalBool(EnableEth1DataVoteCacheFlag.Name) {
		log.Warn("Enabled unsafe eth1 data vote cache")
		cfg.EnableEth1DataVoteCache = true
	}
	if ctx.GlobalBool(EnableSeedCacheFlag.Name) {
		log.Warn("Enabled unsafe seed cache")
		cfg.EnableSeedCache = true
	}
	if ctx.GlobalBool(EnableStartShardCacheFlag.Name) {
		log.Warn("Enabled unsafe start shard cache")
		cfg.EnableStartShardCache = true
	}
	if ctx.GlobalBool(EnableTotalBalanceCacheFlag.Name) {
		log.Warn("Enabled unsafe total balance cache")
		cfg.EnableTotalBalanceCache = true
	}
	InitFeatureConfig(cfg)
}

// ConfigureValidatorFeatures sets the global config based
// on what flags are enabled for the validator client.
func ConfigureValidatorFeatures(ctx *cli.Context) {
	cfg := &FeatureFlagConfig{}
	InitFeatureConfig(cfg)
}

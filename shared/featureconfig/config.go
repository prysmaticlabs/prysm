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
	VerifyAttestationSigs         bool // VerifyAttestationSigs declares if the client will verify attestations.
	EnableComputeStateRoot        bool // EnableComputeStateRoot implementation on server side.
	EnableCrosslinks              bool // EnableCrosslinks in epoch processing.
	EnableCheckBlockStateRoot     bool // EnableCheckBlockStateRoot in block processing.
	DisableHistoricalStatePruning bool // DisableHistoricalStatePruning when updating finalized states.
	DisableGossipSub              bool // DisableGossipSub in p2p messaging.
	EnableCommitteesCache         bool // EnableCommitteesCache for state transition.
	CacheTreeHash                 bool // CacheTreeHash determent whether tree hashes will be cached.
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
	if ctx.GlobalBool(VerifyAttestationSigsFlag.Name) {
		log.Info("Verifying signatures for attestations")
		cfg.VerifyAttestationSigs = true
	}
	if ctx.GlobalBool(EnableComputeStateRootFlag.Name) {
		log.Info("Enabled compute state root server side")
		cfg.EnableComputeStateRoot = true
	}
	if ctx.GlobalBool(EnableCrosslinksFlag.Name) {
		log.Info("Enabled crosslink computations")
		cfg.EnableCrosslinks = true
	}
	if ctx.GlobalBool(EnableCheckBlockStateRootFlag.Name) {
		log.Info("Enabled check block state root")
		cfg.EnableCheckBlockStateRoot = true
	}
	if ctx.GlobalBool(CacheTreeHashFlag.Name) {
		log.Info("Cache tree hashes for ssz")
		cfg.CacheTreeHash = true
	}
	if ctx.GlobalBool(DisableHistoricalStatePruningFlag.Name) {
		log.Info("Enabled historical state pruning")
		cfg.DisableHistoricalStatePruning = true
	}
	if ctx.GlobalBool(DisableGossipSubFlag.Name) {
		log.Info("Disabled gossipsub, using floodsub")
		cfg.DisableGossipSub = true
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
	if ctx.GlobalBool(CacheTreeHashFlag.Name) {
		log.Info("Cache tree hashes for ssz")
		cfg.CacheTreeHash = true
	}

	InitFeatureConfig(cfg)
}

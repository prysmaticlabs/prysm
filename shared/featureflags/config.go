// Package featureflags defines important constants that are essential to the.
// Ethereum 2.0 services.
package featureflags

// FeatureFlagConfig contains configs for the client to determine what it will perform on runtime.
type FeatureFlagConfig struct {
	// Misc constants.
	VerifyAttestationSigs bool // VerifySigs declares if the client will verify attestations
}

var defaultFeatureConfig = &FeatureFlagConfig{
	VerifyAttestationSigs: false,
}

// FeatureConfig retrieves feature config.
func FeatureConfig() *FeatureFlagConfig {
	return featureConfig
}

var featureConfig = defaultFeatureConfig

// DemoFeatureConfig retrieves the demo beacon chain config.
func DemoFeatureConfig() *FeatureFlagConfig {
	demoConfig := *defaultFeatureConfig
	demoConfig.VerifyAttestationSigs = true
	return &demoConfig
}

// UseDemoFeatureConfig for .
func UseDemoFeatureConfig() {
	featureConfig = DemoFeatureConfig()
}

// OverrideFeatureConfig by replacing the config. The preferred pattern is to
// call FeatureConfig(), change the specific parameters, and then call
// OverrideFeatureConfig(c). Any subsequent calls to params.FeatureConfig() will
// return this new configuration.
func OverrideFeatureConfig(c *FeatureFlagConfig) {
	featureConfig = c
}

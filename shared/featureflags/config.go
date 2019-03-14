// Package featureflags defines important constants that are essential to the.
// Ethereum 2.0 services.
package featureflags

// FeatureFlagConfig contains configs for the client to determine what it will perform on runtime.
type FeatureFlagConfig struct {
	// Misc constants.
	VerifyAttestationSigs bool // VerifyAttestationSigs declares if the client will verify attestations.
}

var featureConfig *FeatureFlagConfig

// FeatureConfig retrieves feature config.
func FeatureConfig() *FeatureFlagConfig {
	return featureConfig
}

// InitFeatureConfig retrieves feature config.
func InitFeatureConfig(c *FeatureFlagConfig) {
	featureConfig = c
}

package featureconfig

import (
	"testing"
)

func TestInitFeatureConfig(t *testing.T) {
	cfg := &FeatureFlagConfig{
		VerifyAttestationSigs: true,
	}
	InitFeatureConfig(cfg)
	if c := FeatureConfig(); !c.VerifyAttestationSigs {
		t.Errorf("VerifyAttestationSigs in FeatureFlags incorrect. Wanted true, got false")
	}
}

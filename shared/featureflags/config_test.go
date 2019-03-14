package featureflags

import (
	"testing"
)

func TestOverrideFeatureConfig(t *testing.T) {
	cfg := FeatureConfig()
	cfg.VerifyAttestationSigs = true
	OverrideFeatureConfig(cfg)
	if c := FeatureConfig(); !c.VerifyAttestationSigs {
		t.Errorf("VerifyAttestationSigs in FeatureFlags incorrect. Wanted true, got false")
	}
}

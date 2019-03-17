package featureconfig

import (
	"flag"
	"testing"

	"github.com/urfave/cli"
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

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(VerifyAttestationSigsFlag.Name, true, "enable attestation verification")
	context := cli.NewContext(app, set, nil)
	ConfigureBeaconFeatures(context)
	if c := FeatureConfig(); !c.VerifyAttestationSigs {
		t.Errorf("VerifyAttestationSigs in FeatureFlags incorrect. Wanted true, got false")
	}
}

func TestConfigureValidatorConfig(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(VerifyAttestationSigsFlag.Name, true, "enable attestation verification")
	context := cli.NewContext(app, set, nil)
	ConfigureValidatorFeatures(context)
	if c := FeatureConfig(); !c.VerifyAttestationSigs {
		t.Errorf("VerifyAttestationSigs in FeatureFlags incorrect. Wanted true, got false")
	}
}

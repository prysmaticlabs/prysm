package featureconfig

import (
	"flag"
	"testing"

	"github.com/urfave/cli"
)

func TestInitFeatureConfig(t *testing.T) {
	cfg := &FeatureFlagConfig{
		NoGenesisDelay: true,
	}
	InitFeatureConfig(cfg)
	if c := FeatureConfig(); !c.NoGenesisDelay {
		t.Errorf("NoGenesisDelay in FeatureFlags incorrect. Wanted true, got false")
	}
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(NoGenesisDelayFlag.Name, true, "enable attestation verification")
	context := cli.NewContext(app, set, nil)
	ConfigureBeaconFeatures(context)
	if c := FeatureConfig(); !c.NoGenesisDelay {
		t.Errorf("NoGenesisDelay in FeatureFlags incorrect. Wanted true, got false")
	}
}

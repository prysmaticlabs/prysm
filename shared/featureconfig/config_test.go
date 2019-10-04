package featureconfig_test

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/urfave/cli"
)

func TestInitFeatureConfig(t *testing.T) {
	cfg := &featureconfig.FeatureFlagConfig{
		NoGenesisDelay: true,
	}
	featureconfig.InitFeatureConfig(cfg)
	if c := featureconfig.FeatureConfig(); !c.NoGenesisDelay {
		t.Errorf("NoGenesisDelay in FeatureFlags incorrect. Wanted true, got false")
	}
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(featureconfig.NoGenesisDelayFlag.Name, true, "enable attestation verification")
	context := cli.NewContext(app, set, nil)
	featureconfig.ConfigureBeaconFeatures(context)
	if c := featureconfig.FeatureConfig(); !c.NoGenesisDelay {
		t.Errorf("NoGenesisDelay in FeatureFlags incorrect. Wanted true, got false")
	}
}

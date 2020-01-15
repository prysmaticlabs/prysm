package featureconfig_test

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/urfave/cli"
)

func TestInitFeatureConfig(t *testing.T) {
	cfg := &featureconfig.Flags{
		MinimalConfig: true,
	}
	featureconfig.Init(cfg)
	if c := featureconfig.Get(); !c.MinimalConfig {
		t.Errorf("MinimalConfig in FeatureFlags incorrect. Wanted true, got false")
	}
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(featureconfig.minimalConfigFlag.Name, true, "test")
	context := cli.NewContext(app, set, nil)
	featureconfig.ConfigureBeaconChain(context)
	if c := featureconfig.Get(); !c.MinimalConfig {
		t.Errorf("MinimalConfig in FeatureFlags incorrect. Wanted true, got false")
	}
}

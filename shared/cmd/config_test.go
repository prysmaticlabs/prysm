package cmd

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestInitFeatureConfig(t *testing.T) {
	cfg := &Flags{
		MinimalConfig: true,
	}
	Init(cfg)
	if c := Get(); !c.MinimalConfig {
		t.Errorf("MinimalConfig in cmd flags incorrect. Wanted true, got false")
	}
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool(MinimalConfigFlag.Name, true, "test")
	context := cli.NewContext(&app, set, nil)
	ConfigureBeaconChain(context)
	if c := Get(); !c.MinimalConfig {
		t.Errorf("MinimalConfig in cmd flags incorrect. Wanted true, got false")
	}
}

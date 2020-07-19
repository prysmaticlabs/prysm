package cmd

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/urfave/cli/v2"
)

func TestInitFeatureConfig(t *testing.T) {
	cfg := &Flags{
		MinimalConfig: true,
	}
	Init(cfg)
	c := Get()
	assert.Equal(t, true, c.MinimalConfig)
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool(MinimalConfigFlag.Name, true, "test")
	context := cli.NewContext(&app, set, nil)
	ConfigureBeaconChain(context)
	c := Get()
	assert.Equal(t, true, c.MinimalConfig)
}

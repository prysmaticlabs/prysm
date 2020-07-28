package cmd

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/urfave/cli/v2"
)

func TestOverrideConfig(t *testing.T) {
	cfg := &Flags{
		MinimalConfig: true,
	}
	reset := InitWithReset(cfg)
	defer reset()
	c := Get()
	assert.Equal(t, true, c.MinimalConfig)
}

func TestDefaultConfig(t *testing.T) {
	cfg := &Flags{
		CustomGenesisDelay: params.BeaconConfig().GenesisDelay,
		MaxRPCPageSize:     params.BeaconConfig().DefaultPageSize,
	}
	c := Get()
	assert.DeepEqual(t, c, cfg)

	reset := InitWithReset(cfg)
	defer reset()
	c = Get()
	assert.DeepEqual(t, c, cfg)
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

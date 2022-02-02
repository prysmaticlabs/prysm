package features

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/urfave/cli/v2"
)

func TestInitFeatureConfig(t *testing.T) {
	defer Init(&Flags{})
	cfg := &Flags{
		PyrmontTestnet: true,
	}
	Init(cfg)
	c := Get()
	assert.Equal(t, true, c.PyrmontTestnet)

	// Reset back to false for the follow up tests.
	cfg = &Flags{PyrmontTestnet: false}
	Init(cfg)
}

func TestInitWithReset(t *testing.T) {
	defer Init(&Flags{})
	Init(&Flags{
		PyrmontTestnet: true,
	})
	assert.Equal(t, true, Get().PyrmontTestnet)

	// Overwrite previously set value (value that didn't come by default).
	resetCfg := InitWithReset(&Flags{
		PyrmontTestnet: false,
	})
	assert.Equal(t, false, Get().PyrmontTestnet)

	// Reset must get to previously set configuration (not to default config values).
	resetCfg()
	assert.Equal(t, true, Get().PyrmontTestnet)
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool(PyrmontTestnet.Name, true, "test")
	context := cli.NewContext(&app, set, nil)
	ConfigureBeaconChain(context)
	c := Get()
	assert.Equal(t, true, c.PyrmontTestnet)
}

package featureconfig

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/urfave/cli/v2"
)

func TestInitFeatureConfig(t *testing.T) {
	cfg := &Flags{
		MedallaTestnet: true,
	}
	Init(cfg)
	c := Get()
	assert.Equal(t, true, c.MedallaTestnet)

	// Reset back to false for the follow up tests.
	cfg = &Flags{MedallaTestnet: false}
	Init(cfg)
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool(MedallaTestnet.Name, true, "test")
	context := cli.NewContext(&app, set, nil)
	ConfigureBeaconChain(context)
	c := Get()
	assert.Equal(t, true, c.MedallaTestnet)
}

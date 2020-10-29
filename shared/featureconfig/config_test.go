package featureconfig

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/urfave/cli/v2"
)

func TestInitFeatureConfig(t *testing.T) {
	defer Init(&Flags{})
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

func TestInitWithReset(t *testing.T) {
	defer Init(&Flags{})
	Init(&Flags{
		OnyxTestnet: true,
	})
	assert.Equal(t, false, Get().AltonaTestnet)
	assert.Equal(t, true, Get().OnyxTestnet)

	// Overwrite previously set value (value that didn't come by default).
	resetCfg := InitWithReset(&Flags{
		OnyxTestnet: false,
	})
	assert.Equal(t, false, Get().AltonaTestnet)
	assert.Equal(t, false, Get().OnyxTestnet)

	// Reset must get to previously set configuration (not to default config values).
	resetCfg()
	assert.Equal(t, false, Get().AltonaTestnet)
	assert.Equal(t, true, Get().OnyxTestnet)
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

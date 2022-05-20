package features

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/urfave/cli/v2"
)

func TestInitFeatureConfig(t *testing.T) {
	defer Init(&Flags{})
	cfg := &Flags{
		EnablePeerScorer: true,
	}
	Init(cfg)
	c := Get()
	assert.Equal(t, true, c.EnablePeerScorer)

	// Reset back to false for the follow up tests.
	cfg = &Flags{RemoteSlasherProtection: false}
	Init(cfg)
}

func TestInitWithReset(t *testing.T) {
	defer Init(&Flags{})
	Init(&Flags{
		EnablePeerScorer: true,
	})
	assert.Equal(t, true, Get().EnablePeerScorer)

	// Overwrite previously set value (value that didn't come by default).
	resetCfg := InitWithReset(&Flags{
		EnablePeerScorer: false,
	})
	assert.Equal(t, false, Get().EnablePeerScorer)

	// Reset must get to previously set configuration (not to default config values).
	resetCfg()
	assert.Equal(t, true, Get().EnablePeerScorer)
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool(enablePeerScorer.Name, true, "test")
	context := cli.NewContext(&app, set, nil)
	require.NoError(t, ConfigureBeaconChain(context))
	c := Get()
	assert.Equal(t, true, c.EnablePeerScorer)
}

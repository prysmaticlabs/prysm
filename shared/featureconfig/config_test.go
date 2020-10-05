package featureconfig

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/urfave/cli/v2"
)

func TestInitFeatureConfig(t *testing.T) {
	cfg := &Flags{
		SkipBLSVerify: true,
	}
	Init(cfg)
	c := Get()
	assert.Equal(t, true, c.SkipBLSVerify)

	// Reset back to false for the follow up tests.
	cfg = &Flags{SkipBLSVerify: false}
	Init(cfg)
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool(skipBLSVerifyFlag.Name, true, "test")
	context := cli.NewContext(&app, set, nil)
	ConfigureBeaconChain(context)
	c := Get()
	assert.Equal(t, true, c.SkipBLSVerify)
}

func TestVerifyTestnet(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	context := cli.NewContext(&app, set, nil)
	//temporary warning till next release
	assert.NoError(t, VerifyTestnet(context))

	set.Bool(MedallaTestnet.Name, true, "test")
	context = cli.NewContext(&app, set, nil)
	assert.NoError(t, VerifyTestnet(context))
}

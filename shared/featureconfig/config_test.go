package featureconfig

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestInitFeatureConfig(t *testing.T) {
	cfg := &Flags{
		SkipBLSVerify: true,
	}
	Init(cfg)
	if c := Get(); !c.SkipBLSVerify {
		t.Errorf("SkipBLSVerify in FeatureFlags incorrect. Wanted true, got false")
	}
}

func TestConfigureBeaconConfig(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool(skipBLSVerifyFlag.Name, true, "test")
	context := cli.NewContext(&app, set, nil)
	ConfigureBeaconChain(context)
	if c := Get(); !c.SkipBLSVerify {
		t.Errorf("SkipBLSVerify in FeatureFlags incorrect. Wanted true, got false")
	}
}

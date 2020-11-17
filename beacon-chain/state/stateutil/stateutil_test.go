package stateutil_test

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func TestMain(m *testing.M) {
	run := func() int {
		resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{EnableSSZCache: true})
		defer resetCfg()
		return m.Run()
	}
	os.Exit(run())
}

package cache

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func TestMain(m *testing.M) {
	run := func() int {
		resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{EnableEth1DataVoteCache: true})
		defer resetCfg()
		return m.Run()
	}
	os.Exit(run())
}

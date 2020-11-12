package cache

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func TestMain(m *testing.M) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{EnableEth1DataVoteCache: true})
	defer resetCfg()
	m.Run()
}

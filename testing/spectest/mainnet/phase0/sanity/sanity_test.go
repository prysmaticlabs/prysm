package sanity

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/features"
)

func TestMain(m *testing.M) {
	resetCfg := features.InitWithReset(&features.Flags{EnableBalanceTrieComputation: true})
	defer resetCfg()
	m.Run()
}

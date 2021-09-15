package stateutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/features"
)

func TestMain(m *testing.M) {
	resetCfg := features.InitWithReset(&features.Flags{EnableSSZCache: true})
	defer resetCfg()
	m.Run()
}

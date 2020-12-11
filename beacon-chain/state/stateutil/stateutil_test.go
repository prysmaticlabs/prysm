package stateutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func TestMain(m *testing.M) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{EnableSSZCache: true})
	defer resetCfg()
	m.Run()
}

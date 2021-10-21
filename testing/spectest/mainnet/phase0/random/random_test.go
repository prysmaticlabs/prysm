package random

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/testing/spectest/shared/phase0/sanity"
)

func TestMain(m *testing.M) {
	resetCfg := features.InitWithReset(&features.Flags{EnableBalanceTrieComputation: true})
	defer resetCfg()
	m.Run()
}

func TestMainnet_Phase0_Random(t *testing.T) {
	sanity.RunBlockProcessingTest(t, "mainnet", "random/random/pyspec_tests")
}

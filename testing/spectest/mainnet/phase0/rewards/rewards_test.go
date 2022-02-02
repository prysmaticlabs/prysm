package rewards

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/testing/spectest/shared/phase0/rewards"
)

func TestMain(m *testing.M) {
	resetCfg := features.InitWithReset(&features.Flags{EnableBalanceTrieComputation: true})
	defer resetCfg()
	m.Run()
}

func TestMainnet_Phase0_Rewards(t *testing.T) {
	rewards.RunPrecomputeRewardsAndPenaltiesTests(t, "mainnet")
}

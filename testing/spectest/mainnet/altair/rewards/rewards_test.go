package rewards

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/rewards"
)

func TestMain(m *testing.M) {
	resetCfg := features.InitWithReset(&features.Flags{EnableBalanceTrieComputation: true})
	defer resetCfg()
	m.Run()
}

func TestMainnet_Altair_Rewards(t *testing.T) {
	rewards.RunPrecomputeRewardsAndPenaltiesTests(t, "mainnet")
}

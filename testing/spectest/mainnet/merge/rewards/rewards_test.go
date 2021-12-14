package rewards

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/rewards"
)

func TestMainnet_Merge_Rewards(t *testing.T) {
	rewards.RunPrecomputeRewardsAndPenaltiesTests(t, "mainnet")
}

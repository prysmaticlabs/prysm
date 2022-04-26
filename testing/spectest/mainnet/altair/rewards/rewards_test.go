package rewards

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/rewards"
)

func TestMainnet_Altair_Rewards(t *testing.T) {
	rewards.RunPrecomputeRewardsAndPenaltiesTests(t, "mainnet")
}

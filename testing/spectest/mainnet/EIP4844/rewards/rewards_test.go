package rewards

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/rewards"
)

func TestMainnet_EIP4844_Rewards(t *testing.T) {
	rewards.RunPrecomputeRewardsAndPenaltiesTests(t, "mainnet")
}

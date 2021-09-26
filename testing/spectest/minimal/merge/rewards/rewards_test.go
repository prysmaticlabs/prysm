package rewards

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/rewards"
)

func TestMinimal_Merge_Rewards(t *testing.T) {
	rewards.RunPrecomputeRewardsAndPenaltiesTests(t, "minimal")
}

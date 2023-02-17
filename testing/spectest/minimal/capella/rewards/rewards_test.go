package rewards

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/capella/rewards"
)

func TestMinimal_Capella_Rewards(t *testing.T) {
	rewards.RunPrecomputeRewardsAndPenaltiesTests(t, "minimal")
}

package spectest

import (
	"testing"
)

func TestRewardsPenaltiesMinimal(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runPrecomputeRewardsAndPenaltiesTests(t, "minimal")
}

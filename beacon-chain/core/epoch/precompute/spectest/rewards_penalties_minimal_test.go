package spectest

import (
	"testing"
)

func TestRewardsPenaltiesMinimal(t *testing.T) {
	runPrecomputeRewardsAndPenaltiesTests(t, "minimal")
}

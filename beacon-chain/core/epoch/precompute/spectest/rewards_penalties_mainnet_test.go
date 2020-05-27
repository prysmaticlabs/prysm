package spectest

import (
	"testing"
)

func TestRewardsPenaltiesMainnet(t *testing.T) {
	runPrecomputeRewardsAndPenaltiesTests(t, "mainnet")
}

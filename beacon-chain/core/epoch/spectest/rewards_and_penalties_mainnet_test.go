package spectest

import (
	"testing"
)

func TestRewardsAndPenaltiesMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runRewardsAndPenaltiesTests(t, "mainnet")
}

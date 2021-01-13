package spectest

import (
	"testing"
)

func TestProposerSlashingMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runProposerSlashingTest(t, "mainnet")
}

package spectest

import (
	"testing"
)

func TestProposerSlashingMainnet(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runProposerSlashingTest(t, "mainnet")
}

package spectest

import (
	"testing"
)

func TestProposerSlashingMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")

	runProposerSlashingTest(t, "minimal")
}

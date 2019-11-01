package spectest

import (
	"testing"
)

func TestProposerSlashingMinimal(t *testing.T) {
	runProposerSlashingTest(t, "minimal")
}

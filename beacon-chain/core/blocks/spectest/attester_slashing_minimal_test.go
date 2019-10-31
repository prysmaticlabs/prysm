package spectest

import (
	"testing"
)

func TestAttesterSlashingMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")

	runAttesterSlashingTest(t, "minimal")
}

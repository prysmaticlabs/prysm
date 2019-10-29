package spectest

import (
	"testing"
)

func TestFinalUpdatesMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")

	runFinalUpdatesTests(t, "minimal")
}

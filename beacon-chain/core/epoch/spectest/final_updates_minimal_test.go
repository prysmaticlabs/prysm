package spectest

import (
	"testing"
)

func TestFinalUpdatesMinimal(t *testing.T) {
	t.Skip("Skipping until last stage of 5119")
	runFinalUpdatesTests(t, "minimal")
}

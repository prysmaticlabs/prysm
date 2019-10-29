package spectest

import (
	"testing"
)

func TestSlotProcessingMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runSlotProcessingTests(t, "minimal")
}

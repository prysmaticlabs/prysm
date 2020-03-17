package spectest

import (
	"testing"
)

func TestSlotProcessingMinimal(t *testing.T) {
	t.Skip("Skipping until last stage of 5119")
	runSlotProcessingTests(t, "minimal")
}

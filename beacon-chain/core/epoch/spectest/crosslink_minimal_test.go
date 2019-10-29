package spectest

import (
	"testing"
)

func TestCrosslinksProcessingMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runCrosslinkProcessingTests(t, "minimal")
}

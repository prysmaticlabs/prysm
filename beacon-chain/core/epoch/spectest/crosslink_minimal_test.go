package spectest

import (
	"testing"
)

func TestCrosslinksProcessingMinimal(t *testing.T) {
	runCrosslinkProcessingTests(t, "minimal")
}

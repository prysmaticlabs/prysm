package spectest

import (
	"testing"
)

func TestBlockProcessingMinimal(t *testing.T) {
	runBlockProcessingTest(t, "minimal")
}

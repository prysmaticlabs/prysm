package spectest

import (
	"testing"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	t.Skip("Skip until 3960 merges")
	runBlockProcessingTest(t, "minimal")
}

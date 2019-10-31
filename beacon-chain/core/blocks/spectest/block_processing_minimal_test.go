package spectest

import (
	"testing"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runBlockProcessingTest(t, "minimal")
}

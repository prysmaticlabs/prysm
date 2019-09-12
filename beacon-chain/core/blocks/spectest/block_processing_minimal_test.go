package spectest

import (
	"testing"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	runBlockProcessingTest(t, "minimal")
}

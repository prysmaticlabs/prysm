package spectest

import (
	"testing"
)

func TestBlockHeaderMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")

	runBlockHeaderTest(t, "minimal")
}

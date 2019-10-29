package spectest

import (
	"testing"
)

func TestAttestationMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")

	runAttestationTest(t, "minimal")
}

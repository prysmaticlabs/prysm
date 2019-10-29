package spectest

import (
	"testing"
)

func TestRegistryUpdatesMinimal(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runRegistryUpdatesTests(t, "minimal")
}

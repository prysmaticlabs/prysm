package spectest

import (
	"testing"
)

func TestRegistryUpdatesMinimal(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runRegistryUpdatesTests(t, "minimal")
}

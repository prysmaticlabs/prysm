package spectest

import (
	"testing"
)

func TestRegistryUpdatesMinimal(t *testing.T) {
	t.Skip("Skip until 4272 merged")
	runRegistryUpdatesTests(t, "minimal")
}

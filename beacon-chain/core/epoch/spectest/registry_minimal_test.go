package spectest

import (
	"testing"
)

func TestRegistryUpdatesMinimal(t *testing.T) {
	runRegistryUpdatesTests(t, "minimal")
}

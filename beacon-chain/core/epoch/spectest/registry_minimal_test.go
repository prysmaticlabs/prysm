package spectest

import (
	"testing"
)

func TestRegistryUpdatesMinimal(t *testing.T) {
	t.Skip("Skipping until last stage of 5119")
	runRegistryUpdatesTests(t, "minimal")
}

package spectest

import (
	"testing"
)

func TestRegistryUpdatesMainnet(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runRegistryUpdatesTests(t, "mainnet")
}

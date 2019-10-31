package spectest

import (
	"testing"
)

func TestRegistryUpdatesMainnet(t *testing.T) {
	runRegistryUpdatesTests(t, "mainnet")
}

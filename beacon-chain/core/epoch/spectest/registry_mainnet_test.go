package spectest

import (
	"testing"
)

func TestRegistryUpdatesMainnet(t *testing.T) {
	t.Skip("Skip until 4272 merged")
	runRegistryUpdatesTests(t, "mainnet")
}

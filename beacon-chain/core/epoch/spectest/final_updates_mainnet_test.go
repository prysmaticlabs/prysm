package spectest

import (
	"testing"
)

func TestFinalUpdatesMainnet(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runFinalUpdatesTests(t, "mainnet")
}

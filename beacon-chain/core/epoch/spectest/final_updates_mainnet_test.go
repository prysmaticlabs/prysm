package spectest

import (
	"testing"
)

func TestFinalUpdatesMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runFinalUpdatesTests(t, "mainnet")
}

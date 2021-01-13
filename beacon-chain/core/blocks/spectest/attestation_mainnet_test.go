package spectest

import (
	"testing"
)

func TestAttestationMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runAttestationTest(t, "mainnet")
}

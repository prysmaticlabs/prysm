package spectest

import (
	"testing"
)

func TestAttestationMainnet(t *testing.T) {
	t.Skip("Skip until 3960 merges")
	runAttestationTest(t, "mainnet")
}

package spectest

import (
	"testing"
)

func TestAttesterSlashingMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runAttesterSlashingTest(t, "mainnet")
}

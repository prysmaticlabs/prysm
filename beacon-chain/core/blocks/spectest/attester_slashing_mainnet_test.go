package spectest

import (
	"testing"
)

func TestAttesterSlashingMainnet(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runAttesterSlashingTest(t, "mainnet")
}

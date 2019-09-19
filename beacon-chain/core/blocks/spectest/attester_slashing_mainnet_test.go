package spectest

import (
	"testing"
)

func TestAttesterSlashingMainnet(t *testing.T) {
	runAttesterSlashingTest(t, "mainnet")
}

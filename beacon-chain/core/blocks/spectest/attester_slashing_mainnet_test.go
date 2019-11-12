package spectest

import (
	"testing"
)

func TestAttesterSlashingMainnet(t *testing.T) {
	t.Skip("Skip until 3960 merges")
	runAttesterSlashingTest(t, "mainnet")
}

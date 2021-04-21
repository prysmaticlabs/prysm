package spectest

import "testing"

func TestEffectiveBalanceUpdatesMinimal(t *testing.T) {
	runEffectiveBalanceUpdatesTests(t, "minimal")
}

package spectest

import "testing"

func TestEffectiveBalanceUpdatesMainnet(t *testing.T) {
	runEffectiveBalanceUpdatesTests(t, "mainnet")
}

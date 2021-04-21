package spectest

import "testing"

func TestRandaoMixesResetMainnet(t *testing.T) {
	runRandaoMixesResetTests(t, "mainnet")
}

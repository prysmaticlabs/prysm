package spectest

import (
	"testing"
)

func TestDepositMainnetYaml(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runDepositTest(t, "mainnet")
}

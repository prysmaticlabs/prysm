package spectest

import (
	"testing"
)

func TestBlockHeaderMainnet(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runBlockHeaderTest(t, "mainnet")
}

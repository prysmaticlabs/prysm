package spectest

import (
	"testing"
)

func TestBlockHeaderMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runBlockHeaderTest(t, "mainnet")
}

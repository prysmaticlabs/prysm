package spectest

import (
	"testing"
)

func TestBlockHeaderMainnet(t *testing.T) {
	runBlockHeaderTest(t, "mainnet")
}

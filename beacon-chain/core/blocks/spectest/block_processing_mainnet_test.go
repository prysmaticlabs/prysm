package spectest

import (
	"testing"
)

func TestBlockProcessingMainnet(t *testing.T) {
	runBlockProcessingTest(t, "mainnet")
}

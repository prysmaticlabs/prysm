package spectest

import (
	"testing"
)

func TestCrosslinksProcessingMainnet(t *testing.T) {
	runCrosslinkProcessingTests(t, "mainnet")
}

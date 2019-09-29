package spectest

import (
	"testing"
)

func TestSlotProcessingMainnet(t *testing.T) {
	runSlotProcessingTests(t, "mainnet")
}

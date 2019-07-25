package spectest

import (
	"testing"
)

func TestSlotProcessingMainnet(t *testing.T) {
	runSlotProcessingTests(t, "sanity_slots_mainnet.yaml")
}

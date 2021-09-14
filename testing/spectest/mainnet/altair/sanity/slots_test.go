package sanity

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/sanity"
)

func TestMainnet_Altair_Sanity_Slots(t *testing.T) {
	sanity.RunSlotProcessingTests(t, "mainnet")
}

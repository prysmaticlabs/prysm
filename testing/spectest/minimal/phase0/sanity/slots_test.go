package sanity

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/phase0/sanity"
)

func TestMinimal_Phase0_Sanity_Slots(t *testing.T) {
	sanity.RunSlotProcessingTests(t, "minimal")
}

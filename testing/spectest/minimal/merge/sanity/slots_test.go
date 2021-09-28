package sanity

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/sanity"
)

func TestMinimal_Merge_Sanity_Slots(t *testing.T) {
	sanity.RunSlotProcessingTests(t, "minimal")
}

package sanity

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/sanity"
)

func TestMinimal_Merge_Sanity_Blocks(t *testing.T) {
	sanity.RunBlockProcessingTest(t, "minimal", "sanity/blocks/pyspec_tests")
}

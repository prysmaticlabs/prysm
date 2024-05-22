package sanity

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/sanity"
)

func TestMinimal_Electra_Sanity_Blocks(t *testing.T) {
	t.Skip("TODO: Electra")
	sanity.RunBlockProcessingTest(t, "minimal", "sanity/blocks/pyspec_tests")
}

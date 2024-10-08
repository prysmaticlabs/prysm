package sanity

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/sanity"
)

func TestMainnet_Electra_Sanity_Blocks(t *testing.T) {
	t.Skip("TODO: add back in after all spec test features are in.")
	sanity.RunBlockProcessingTest(t, "mainnet", "sanity/blocks/pyspec_tests")
}

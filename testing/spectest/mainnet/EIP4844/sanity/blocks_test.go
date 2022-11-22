package sanity

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/sanity"
)

func TestMainnet_EIP4844_Sanity_Blocks(t *testing.T) {
	sanity.RunBlockProcessingTest(t, "mainnet", "sanity/blocks/pyspec_tests")
}

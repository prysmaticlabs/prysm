package random

import (
	"testing"

	"github.com/prysmaticlabs/prysm/spectest/shared/altair/sanity"
)

func TestMainnet_Altair_Random(t *testing.T) {
	sanity.RunBlockProcessingTest(t, "mainnet", "random/random/pyspec_tests")
}

package random

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/bellatrix/sanity"
)

func TestMinimal_Bellatrix_Random(t *testing.T) {
	t.Skip("Test is not available: https://github.com/ethereum/consensus-spec-tests/tree/master/tests/minimal/bellatrix")
	sanity.RunBlockProcessingTest(t, "minimal", "random/random/pyspec_tests")
}

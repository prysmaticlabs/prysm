package random

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/sanity"
)

func TestMinimal_Merge_Random(t *testing.T) {
	t.Skip("Test is not available: https://github.com/ethereum/consensus-spec-tests/tree/master/tests/minimal/merge")
	sanity.RunBlockProcessingTest(t, "minimal", "random/random/pyspec_tests")
}

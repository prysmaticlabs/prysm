package random

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/sanity"
)

func TestMinimal_Merge_Random(t *testing.T) {
	sanity.RunBlockProcessingTest(t, "minimal", "random/random/pyspec_tests")
}

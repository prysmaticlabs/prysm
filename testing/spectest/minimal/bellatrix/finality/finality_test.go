package finality

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/bellatrix/finality"
)

func TestMinimal_Merge_Finality(t *testing.T) {
	finality.RunFinalityTest(t, "minimal")
}

package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/operations"
)

func TestMinimal_Electra_Operations_Consolidation(t *testing.T) {
	t.Skip("TODO: add back in after all spec test features are in.")
	operations.RunConsolidationTest(t, "minimal")
}

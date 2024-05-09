package finality

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/finality"
)

func TestMinimal_Electra_Finality(t *testing.T) {
	t.Skip("TODO: Electra")
	finality.RunFinalityTest(t, "minimal")
}

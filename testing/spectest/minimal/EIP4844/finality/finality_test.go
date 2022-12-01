package finality

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/finality"
)

func TestMinimal_EIP4844_Finality(t *testing.T) {
	finality.RunFinalityTest(t, "minimal")
}

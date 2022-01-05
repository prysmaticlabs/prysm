package fork

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/bellatrix/fork"
)

func TestMinimal_Merge_UpgradeToMerge(t *testing.T) {
	t.Skip("Test is not available: https://github.com/ethereum/consensus-spec-tests/tree/master/tests/minimal/bellatrix")
	fork.RunUpgradeToMerge(t, "minimal")
}

package fork

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/fork"
)

func TestMinimal_Merge_UpgradeToMerge(t *testing.T) {
	fork.RunUpgradeToMerge(t, "minimal")
}

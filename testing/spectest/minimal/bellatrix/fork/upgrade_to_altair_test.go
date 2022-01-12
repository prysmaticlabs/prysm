package fork

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/bellatrix/fork"
)

func TestMinimal_Bellatrix_UpgradeToMerge(t *testing.T) {
	fork.RunUpgradeToBellatrix(t, "minimal")
}

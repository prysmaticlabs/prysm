package fork

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/bellatrix/fork"
)

func TestMinimal_Bellatrix_UpgradeToBellatrix(t *testing.T) {
	fork.RunUpgradeToBellatrix(t, "minimal")
}

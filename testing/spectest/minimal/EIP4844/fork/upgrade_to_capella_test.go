package fork

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/fork"
)

func TestMinimal_EIP4844_UpgradeToEIP4844(t *testing.T) {
	fork.RunUpgradeTo48444(t, "minimal")
}

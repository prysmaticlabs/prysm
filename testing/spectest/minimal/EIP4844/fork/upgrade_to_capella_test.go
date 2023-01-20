package fork

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/deneb/fork"
)

func TestMinimal_Deneb_UpgradeToDeneb(t *testing.T) {
	fork.RunUpgradeToDeneb4(t, "minimal")
}

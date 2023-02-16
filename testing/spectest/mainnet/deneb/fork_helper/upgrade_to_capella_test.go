package fork_helper

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/deneb/fork"
)

func TestMainnet_Deneb_UpgradeToCapella(t *testing.T) {
	fork.RunUpgradeToDeneb4(t, "mainnet")
}

package fork_helper

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/fork"
)

func TestMainnet_EIP4844_UpgradeToCapella(t *testing.T) {
	fork.RunUpgradeToCapella(t, "mainnet")
}

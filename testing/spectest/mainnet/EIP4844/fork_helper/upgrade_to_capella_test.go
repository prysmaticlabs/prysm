package fork_helper

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/fork"
)

func TestMainnet_EIP4844_UpgradeToEIP4844(t *testing.T) {
	fork.RunUpgradeToCapella(t, "mainnet")
}

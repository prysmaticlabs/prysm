package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/epoch_processing"
)

func TestMainnet_EIP4844_EpochProcessing_ResetRegistryUpdates(t *testing.T) {
	epoch_processing.RunRegistryUpdatesTests(t, "mainnet")
}

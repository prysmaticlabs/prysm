package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/spectest/shared/phase0/epoch_processing"
)

func TestMainnet_Phase0_EpochProcessing_ResetRegistryUpdates(t *testing.T) {
	epoch_processing.RunRegistryUpdatesTests(t, "mainnet")
}

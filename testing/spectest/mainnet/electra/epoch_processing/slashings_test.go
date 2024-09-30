package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/epoch_processing"
)

func TestMainnet_Electra_EpochProcessing_Slashings(t *testing.T) {
	t.Skip("slashing processing missing")
	epoch_processing.RunSlashingsTests(t, "mainnet")
}

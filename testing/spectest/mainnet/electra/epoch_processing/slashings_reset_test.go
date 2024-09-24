package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/epoch_processing"
)

func TestMainnet_Electra_EpochProcessing_SlashingsReset(t *testing.T) {
	t.Skip("slashing processing missing")
	epoch_processing.RunSlashingsResetTests(t, "mainnet")
}

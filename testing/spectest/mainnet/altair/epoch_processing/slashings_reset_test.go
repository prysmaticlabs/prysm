package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/epoch_processing"
)

func TestMainnet_Altair_EpochProcessing_SlashingsReset(t *testing.T) {
	epoch_processing.RunSlashingsResetTests(t, "mainnet")
}

package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/epoch_processing"
)

func TestMainnet_Altair_EpochProcessing_JustificationAndFinalization(t *testing.T) {
	epoch_processing.RunJustificationAndFinalizationTests(t, "mainnet")
}

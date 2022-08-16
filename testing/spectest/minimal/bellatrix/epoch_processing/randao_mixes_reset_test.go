package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/bellatrix/epoch_processing"
)

func TestMinimal_Bellatrix_EpochProcessing_RandaoMixesReset(t *testing.T) {
	epoch_processing.RunRandaoMixesResetTests(t, "minimal")
}

package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/testing/spectest/shared/phase0/epoch_processing"
)

func TestMinimal_Phase0_EpochProcessing_JustificationAndFinalization(t *testing.T) {
	epoch_processing.RunJustificationAndFinalizationTests(t, "minimal")
}

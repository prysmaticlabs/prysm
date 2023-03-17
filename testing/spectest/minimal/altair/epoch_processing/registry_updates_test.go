package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/testing/spectest/shared/altair/epoch_processing"
)

func TestMinimal_Altair_EpochProcessing_ResetRegistryUpdates(t *testing.T) {
	epoch_processing.RunRegistryUpdatesTests(t, "minimal")
}

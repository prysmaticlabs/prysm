package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/epoch_processing"
)

func TestMinimal_Electra_EpochProcessing_PendingConsolidations(t *testing.T) {
	t.Skip("TODO: add back in after all spec test features are in.")
	epoch_processing.RunPendingConsolidationsTests(t, "minimal")
}

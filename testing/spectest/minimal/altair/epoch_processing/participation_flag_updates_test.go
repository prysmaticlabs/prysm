package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/testing/spectest/shared/altair/epoch_processing"
)

func TestMinimal_Altair_EpochProcessing_ParticipationFlag(t *testing.T) {
	epoch_processing.RunParticipationFlagUpdatesTests(t, "minimal")
}

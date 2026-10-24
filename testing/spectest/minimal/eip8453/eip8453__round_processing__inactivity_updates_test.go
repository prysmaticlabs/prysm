package eip8453

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/testing/spectest/shared/eip8453/epoch_processing"
)

func TestMinimal_EIP8453_RoundProcessing_InactivityUpdates(t *testing.T) {
	epoch_processing.RunInactivityUpdatesTest(t, "minimal")
}

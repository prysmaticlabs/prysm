package epoch_processing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/deneb/epoch_processing"
)

func TestMainnet_Deneb_EpochProcessing_EffectiveBalanceUpdates(t *testing.T) {
	epoch_processing.RunEffectiveBalanceUpdatesTests(t, "mainnet")
}

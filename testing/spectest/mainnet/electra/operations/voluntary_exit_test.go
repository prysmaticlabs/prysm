package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/operations"
)

func TestMainnet_Electra_Operations_VoluntaryExit(t *testing.T) {
	t.Skip("TODO: Electra")
	operations.RunVoluntaryExitTest(t, "mainnet")
}

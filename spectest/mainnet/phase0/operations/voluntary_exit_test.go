package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/spectest/shared/phase0/operations"
)

func TestMainnet_Phase0_Operations_VoluntaryExit(t *testing.T) {
	operations.RunVoluntaryExitTest(t, "mainnet")
}

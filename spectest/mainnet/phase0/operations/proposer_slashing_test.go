package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/spectest/shared/phase0/operations"
)

func TestMainnet_Phase0_Operations_ProposerSlashing(t *testing.T) {
	operations.RunProposerSlashingTest(t, "mainnet")
}

package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/operations"
)

func TestMainnet_Altair_Operations_AttesterSlashing(t *testing.T) {
	operations.RunAttesterSlashingTest(t, "mainnet")
}

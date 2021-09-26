package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/operations"
)

func TestMainnet_Merge_Operations_AttesterSlashing(t *testing.T) {
	operations.RunAttesterSlashingTest(t, "mainnet")
}

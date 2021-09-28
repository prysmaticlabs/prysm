package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/operations"
)

func TestMainnet_Merge_Operations_Attestation(t *testing.T) {
	operations.RunAttestationTest(t, "mainnet")
}

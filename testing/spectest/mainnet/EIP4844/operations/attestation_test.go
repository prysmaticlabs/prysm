package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/operations"
)

func TestMainnet_EIP4844_Operations_Attestation(t *testing.T) {
	operations.RunAttestationTest(t, "mainnet")
}

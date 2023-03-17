package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/testing/spectest/shared/phase0/operations"
)

func TestMinimal_Phase0_Operations_Attestation(t *testing.T) {
	operations.RunAttestationTest(t, "minimal")
}

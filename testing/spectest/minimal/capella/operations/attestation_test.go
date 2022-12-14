package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/capella/operations"
)

func TestMinimal_Capella_Operations_Attestation(t *testing.T) {
	operations.RunAttestationTest(t, "minimal")
}

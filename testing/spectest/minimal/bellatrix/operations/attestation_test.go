package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/bellatrix/operations"
)

func TestMinimal_Bellatrix_Operations_Attestation(t *testing.T) {
	operations.RunAttestationTest(t, "minimal")
}

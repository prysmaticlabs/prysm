package spectest

import (
	"testing"
)

func TestAttestationMinimal(t *testing.T) {
	runAttestationTest(t, "attestation_minimal.yaml")
}

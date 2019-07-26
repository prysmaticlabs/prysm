package spectest

import (
	"testing"
)

func TestAttestationMainnet(t *testing.T) {
	runAttestationTest(t, "attestation_mainnet.yaml")
}

package attestations

import (
	"testing"
)

func TestContainsValidator(t *testing.T) {
	if !ContainsValidator([]byte{7}, []byte{4}) {
		t.Error("Attestation should contain validator")
	}

	if ContainsValidator([]byte{7}, []byte{8}) {
		t.Error("Attestation should not contain validator")
	}
}

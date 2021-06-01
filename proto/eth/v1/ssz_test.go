package v1

import (
	"io/ioutil"
	"testing"

	"github.com/ferranbt/fastssz"
)

func TestMisalignedAttestationOffset(t *testing.T) {
	buf, err := ioutil.ReadFile("testdata/invalid-offset.attestation.ssz")
	if err != nil {
		t.Error(err)
	}
	a := Attestation{}
	err = a.UnmarshalSSZ(buf)
	if err != ssz.ErrInvalidVariableOffset {
		t.Errorf("Expected error of type ssz.ErrInvalidVariableOffset, got = %v", err)
	}
}

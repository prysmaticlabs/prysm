package attestations

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestContainsValidator(t *testing.T) {
	if !ContainsValidator([]byte{7}, []byte{4}) {
		t.Error("Attestation should contain validator")
	}

	if ContainsValidator([]byte{7}, []byte{8}) {
		t.Error("Attestation should not contain validator")
	}
}

func TestIsDoubleVote(t *testing.T) {
	att1 := &pb.AttestationData{
		Slot: 0,
	}

	att2 := &pb.AttestationData{
		Slot: 64,
	}

	if IsDoubleVote(att1, att2) {
		t.Error("It is a double vote despite the attestations being on different epochs")
	}

	att2.Slot = 1

	if !IsDoubleVote(att1, att2) {
		t.Error("It is not a double vote despite the attestations being on the same epoch")
	}
}

func TestIsSurroundVote(t *testing.T) {
	att1 := &pb.AttestationData{
		Slot:          0,
		JustifiedSlot: 0,
	}

	att2 := &pb.AttestationData{
		Slot:          0,
		JustifiedSlot: 0,
	}

	if IsSurroundVote(att1, att2) {
		t.Error("It is a surround vote despite both attestations having the same epoch")
	}

	if IsSurroundVote(att1, att2) {
		t.Error("It is a surround vote despite both attestations being on the same epoch")
	}

	att1.Slot = 192
	att2.JustifiedSlot = 64
	att2.Slot = 128

	if !IsSurroundVote(att1, att2) {
		t.Error("It is not a surround vote despite all the surround conditions being fulfilled")
	}

}

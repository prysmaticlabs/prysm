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
		Slot: 1,
	}

	att2 := &pb.AttestationData{
		Slot: 2,
	}

	if IsDoubleVote(att1, att2) {
		t.Error("It is a double vote despite the attestations being on different slots")
	}

	att2.Slot = 1

	if !IsDoubleVote(att1, att2) {
		t.Error("It is not a double vote despite the attestations being on the same slot")
	}
}

func TestIsSurroundVote(t *testing.T) {
	att1 := &pb.AttestationData{
		Slot:          2,
		JustifiedSlot: 1,
	}

	att2 := &pb.AttestationData{
		Slot:          1,
		JustifiedSlot: 1,
	}

	if IsSurroundVote(att1, att2) {
		t.Error("It is a surround vote despite both attestations having the same justified slot")
	}

	att2.Slot++

	if IsSurroundVote(att1, att2) {
		t.Error("It is a surround vote despite both attestations having the same slot.")
	}

	att1.Slot = 4
	att2.JustifiedSlot = 2
	att2.Slot++

	if !IsSurroundVote(att1, att2) {
		t.Error("It is not a surround vote despite all the surround conditions being fulfilled")
	}

}

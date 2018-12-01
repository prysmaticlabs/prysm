package types

import (
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestAttestation(t *testing.T) {
	data := &pb.AggregatedAttestation{
		SignedData: &pb.AttestationSignedData{
			Slot:               0,
			Shard:              0,
			JustifiedSlot:      0,
			JustifiedBlockHash: []byte{0},
			ShardBlockHash:     []byte{0},
		},
		AttesterBitfield: []byte{0},
		AggregateSig:     []uint64{0},
	}
	data2 := &pb.ProcessedAttestation{
		SignedData: &pb.AttestationSignedData{
			Slot:               0,
			Shard:              0,
			JustifiedSlot:      0,
			JustifiedBlockHash: []byte{0},
			ShardBlockHash:     []byte{0},
		},
		SlotIncluded: 1,
	}
	attestation := NewAggregatedAttestation(data)
	attestation2 := NewProcessedAttestation(data2)
	attestation.AttesterBitfield()
	attestation.AggregateSig()
	attestation.SignedData()
	attestation2.SlotIncluded()

	emptyAttestation := &AggregatedAttestation{}
	if _, err := emptyAttestation.Marshal(); err == nil {
		t.Error("marshal with empty data should fail")
	}
	if _, err := emptyAttestation.Hash(); err == nil {
		t.Error("hash with empty data should fail")
	}
	if _, err := attestation.Hash(); err != nil {
		t.Errorf("hashing with data should not fail, received %v", err)
	}
	if !reflect.DeepEqual(attestation.data, attestation.Proto()) {
		t.Errorf("inner block data did not match proto: received %v, wanted %v", attestation.Proto(), attestation.data)
	}
	if attestation.SignedData().GetSlot() != 0 {
		t.Errorf("mismatched attestation slot number: wanted 0, received %v", attestation.SignedData().GetSlot())
	}
	if err := VerifyProposerAttestation(attestation.SignedData(), [32]byte{}, 0); err != nil {
		t.Errorf("verify attestation failed: %v", err)
	}
}

func TestContainsValidator(t *testing.T) {
	attestation := NewAggregatedAttestation(&pb.AggregatedAttestation{
		SignedData: &pb.AttestationSignedData{
			Slot:  0,
			Shard: 0,
		},
		AttesterBitfield: []byte{7}, // 0000 0111
	})

	if !ContainsValidator(attestation.AttesterBitfield(), []byte{4}) {
		t.Error("Attestation should contain validator")
	}

	if ContainsValidator(attestation.AttesterBitfield(), []byte{8}) {
		t.Error("Attestation should not contain validator")
	}
}

package types

import (
	"bytes"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestAttestation(t *testing.T) {
	data := &pb.AggregatedAttestation{
		Slot:                0,
		Shard:               0,
		JustifiedSlot:       0,
		JustifiedBlockHash:  []byte{0},
		ShardBlockHash:      []byte{0},
		AttesterBitfield:    []byte{0},
		ObliqueParentHashes: [][]byte{{0}},
		AggregateSig:        []uint64{0},
	}
	attestation := NewAttestation(data)
	attestation.SlotNumber()
	attestation.ShardID()
	attestation.JustifiedSlotNumber()
	attestation.JustifiedBlockHash()
	attestation.AttesterBitfield()
	attestation.ObliqueParentHashes()
	attestation.AggregateSig()
	attestation.Key()

	emptyAttestation := &Attestation{}
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
	if attestation.SlotNumber() != 0 {
		t.Errorf("mismatched attestation slot number: wanted 0, received %v", attestation.SlotNumber())
	}
	attestationWithNilData := NewAttestation(nil)
	if attestationWithNilData.ShardID() != 0 {
		t.Errorf("mismatched attestation shard id: wanted 0, received %v", attestation.ShardID())
	}
	if !bytes.Equal(attestation.ShardBlockHash(), []byte{0}) {
		t.Errorf("mismatched shard block hash")
	}
	if err := attestation.VerifyProposerAttestation([32]byte{}, 0); err != nil {
		t.Errorf("verify attestation failed: %v", err)
	}
}

func TestContainsValidator(t *testing.T) {
	attestation := NewAttestation(&pb.AggregatedAttestation{
		Slot:             0,
		Shard:            0,
		AttesterBitfield: []byte{7}, // 0000 0111
	})

	if !attestation.ContainsValidator([]byte{4}) {
		t.Error("Attestation should contain validator")
	}

	if attestation.ContainsValidator([]byte{8}) {
		t.Error("Attestation should not contain validator")
	}
}

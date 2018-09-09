package types

import (
	"bytes"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestAttestation(t *testing.T) {
	data := &pb.AttestationRecord{
		Slot:                0,
		ShardId:             0,
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
}

package types

import (
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestBlock(t *testing.T) {
	data := &pb.BeaconBlock{
		ParentHash:            []byte{0},
		SlotNumber:            0,
		RandaoReveal:          []byte{0},
		Attestations:          []*pb.AttestationRecord{},
		PowChainRef:           []byte{0},
		ActiveStateHash:       []byte{0},
		CrystallizedStateHash: []byte{0},
	}
	block := &Block{data}
	block.ParentHash()
	block.PowChainRef()
	block.RandaoReveal()
	block.ActiveStateHash()
	block.CrystallizedStateHash()
	newBlockNil := NewBlock(nil)
	block.data.Timestamp = newBlockNil.data.Timestamp
	if !reflect.DeepEqual(block, newBlockNil) {
		t.Errorf("mismatched blocks: wanted %v, received %v", block, newBlockNil)
	}
	data = &pb.BeaconBlock{SlotNumber: 1}
	block = &Block{data}
	block.data.Timestamp = newBlockNil.data.Timestamp
	if !reflect.DeepEqual(block, NewBlock(data)) {
		t.Errorf("mismatched blocks: wanted %v, received %v", block, NewBlock(data))
	}
	emptyBlock := &Block{}
	if _, err := emptyBlock.Marshal(); err == nil {
		t.Error("marshal with empty data should fail")
	}
	if _, err := emptyBlock.Hash(); err == nil {
		t.Error("hash with empty data should fail")
	}
	if _, err := block.Timestamp(); err != nil {
		t.Errorf("well formatted timestamp should not throw an error, received %v", err)
	}
	if _, err := block.Hash(); err != nil {
		t.Errorf("hashing with data should not fail, received %v", err)
	}
	if !reflect.DeepEqual(block.data, block.Proto()) {
		t.Errorf("inner block data did not match proto: received %v, wanted %v", block.Proto(), block.data)
	}
	if block.AttestationCount() != 0 {
		t.Errorf("mismatched attestation count: wanted 0, received %v", block.AttestationCount())
	}
	if _, err := NewGenesisBlock(); err != nil {
		t.Errorf("creating genesis block should not throw error, received %v", err)
	}
}

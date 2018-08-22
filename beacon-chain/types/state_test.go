package types

import (
	"bytes"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestBlockHashForSlot(t *testing.T) {
	var recentBlockHash [][]byte
	for i := 0; i < 256; i++ {
		recentBlockHash = append(recentBlockHash, []byte{byte(i)})
	}
	state := NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentBlockHash,
	})
	block := newTestBlock(t, &pb.BeaconBlock{SlotNumber: 7})
	if _, err := state.BlockHashForSlot(200, block); err == nil {
		t.Error("getBlockHash should have failed with invalid height")
	}
	hash, err := state.BlockHashForSlot(0, block)
	if err != nil {
		t.Errorf("getBlockHash failed: %v", err)
	}
	if bytes.Equal(hash, []byte{'A'}) {
		t.Errorf("getBlockHash returns hash should be A, got: %v", hash)
	}
	hash, err = state.BlockHashForSlot(5, block)
	if err != nil {
		t.Errorf("getBlockHash failed: %v", err)
	}
	if bytes.Equal(hash, []byte{'F'}) {
		t.Errorf("getBlockHash returns hash should be F, got: %v", hash)
	}
	block = newTestBlock(t, &pb.BeaconBlock{SlotNumber: 201})
	hash, err = state.BlockHashForSlot(200, block)
	if err != nil {
		t.Errorf("getBlockHash failed: %v", err)
	}
	if hash[len(hash)-1] != 127 {
		t.Errorf("getBlockHash returns hash should be 127, got: %v", hash)
	}

}

// newTestBlock is a helper method to create blocks with valid defaults.
// For a generic block, use NewBlock(t, nil).
func newTestBlock(t *testing.T, b *pb.BeaconBlock) *Block {
	if b == nil {
		b = &pb.BeaconBlock{}
	}
	if b.ActiveStateHash == nil {
		b.ActiveStateHash = make([]byte, 32)
	}
	if b.CrystallizedStateHash == nil {
		b.CrystallizedStateHash = make([]byte, 32)
	}
	if b.ParentHash == nil {
		b.ParentHash = make([]byte, 32)
	}

	return NewBlock(b)
}

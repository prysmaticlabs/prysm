package types

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestBlockHashForSlot(t *testing.T) {
	state := NewActiveState(&pb.ActiveState{
		RecentBlockHashes: [][]byte{
			{'A'},
			{'B'},
			{'C'},
			{'D'},
			{'E'},
			{'F'},
		},
	})
	if _, err := state.BlockHashForSlot(200, 250); err == nil {
		t.Error("should have failed with invalid height")
	}
	hash, err := state.BlockHashForSlot(2*params.CycleLength, 0)
	if err != nil {
		t.Errorf("BlockHashForSlot failed: %v", err)
	}
	if bytes.Equal(hash, []byte{'A'}) {
		t.Errorf("BlockHashForSlot returns hash should be A, got: %v", hash)
	}
	hash, err = state.BlockHashForSlot(2*params.CycleLength, uint64(len(state.RecentBlockHashes())-1))
	if err != nil {
		t.Errorf("BlockHashForSlot failed: %v", err)
	}
	if bytes.Equal(hash, []byte{'F'}) {
		t.Errorf("BlockHashForSlot returns hash should be F, got: %v", hash)
	}
}

package types

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/ptypes"
	"github.com/prysmaticlabs/prysm/shared/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGenesisBlock(t *testing.T) {
	stateHash := [32]byte{0}
	b1 := NewGenesisBlock(stateHash)
	b2 := NewGenesisBlock(stateHash)

	// We ensure that initializing a proto timestamp from
	// genesis time will lead to no error.
	if _, err := ptypes.TimestampProto(params.BeaconConfig().GenesisTime); err != nil {
		t.Errorf("could not create proto timestamp, expected no error: %v", err)
	}

	h1, err1 := b1.Hash()
	h2, err2 := b2.Hash()
	if err1 != nil || err2 != nil {
		t.Fatalf("failed to hash genesis block: %v %v", err1, err2)
	}

	if h1 != h2 {
		t.Errorf("genesis block hash should be identical: %#x %#x", h1, h2)
	}

	if b1.data.ParentRootHash32 == nil {
		t.Error("genesis block missing ParentHash field")
	}

	if b1.AttestationCount() > 0 || b1.Attestations() != nil {
		t.Errorf("genesis block should have 0 attestations")
	}

	if b1.RandaoRevealHash32() != [32]byte{} {
		t.Error("genesis block missing RandaoRevealHash32 field")
	}

	if b1.CandidatePowReceiptRootHash32() != common.BytesToHash([]byte{}) {
		t.Error("genesis block missing CandidatePowReceiptRootHash32 field")
	}

	if b1.StateRootHash32() != stateHash {
		t.Error("genesis block StateRootHash32 isn't initialized correctly")
	}

	rd := [32]byte{}
	if b1.IsRandaoValid(rd[:]) {
		t.Error("RANDAO should be empty")
	}

	gt1, _ := b1.Timestamp()
	gt2, _ := b2.Timestamp()
	if gt1 != gt2 {
		t.Error("different timestamp")
	}

	enc1, _ := b1.Marshal()
	enc2, _ := b2.Marshal()
	if !bytes.Equal(enc1, enc2) {
		t.Error("genesis block encoding does not match")
	}

	if !reflect.DeepEqual(b1.Proto(), b2.Proto()) {
		t.Error("genesis blocks proto should be equal")
	}

	b3 := NewBlock(nil)
	h3, err3 := b3.Hash()
	if err3 != nil {
		t.Fatalf("failed to hash genesis block: %v", err3)
	}

	if h1 == h3 {
		t.Errorf("genesis block and new block should not have identical hash: %#x", h1)
	}
}

func TestBlockRootAtSlot_OK(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	state := &pb.BeaconState{
		LatestBlockRootHash32S: blockRoots,
	}

	tests := []struct {
		slot          uint64
		stateSlot     uint64
		expectedRoot  []byte
	}{
		{
			slot:          0,
			stateSlot:     0,
			expectedRoot: 0,
		},
		{
			slot:          1,
			stateSlot:     5,
			expectedRoot: 1,
		},
		{
			stateSlot:     1024,
			slot:          1024,
			expectedRoot: 64 - 0,
		}, {
			stateSlot:     2048,
			slot:          2000,
			expectedRoot: 64 - 48,
		}, {
			stateSlot:     2048,
			slot:          2058,
			expectedRoot: 64 + 10,
		},
	}
	for _, tt := range tests {
		state.Slot = tt.stateSlot
		result, err := ShardAndCommitteesAtSlot(state, tt.slot)
		if err != nil {
			t.Errorf("Failed to get shard and committees at slot: %v", err)
		}
		if result.ArrayShardAndCommittee[0].Shard != tt.expectedShard {
			t.Errorf(
				"Result shard was an unexpected value. Wanted %d, got %d",
				tt.expectedShard,
				result.ArrayShardAndCommittee[0].Shard,
			)
		}
	}
}

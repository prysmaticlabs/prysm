package utils

import (
	"reflect"
	"testing"
)

func TestBlockVoteMarshalUnmarshall(t *testing.T) {
	v1 := NewBlockVote()
	v1.VoterIndices = []uint32{1, 2, 3}
	v1.VoteTotalDeposit = 10

	blob, err := v1.Marshal()
	if err != nil {
		t.Fatalf("fail to serialize block vote: %v", err)
	}

	v2 := new(BlockVote)
	if err = v2.Unmarshal(blob); err != nil {
		t.Fatalf("fail to deserialize block vote: %v", err)
	}

	if !reflect.DeepEqual(v1, v2) {
		t.Error("block vote cache serialization and deserialization don't match")
	}
}

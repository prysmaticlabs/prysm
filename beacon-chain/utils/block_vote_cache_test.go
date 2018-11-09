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
		t.Error("Fail to serialize BlockVote")
	}

	v2 := new(BlockVote)
	if err = v2.Unmarshal(blob); err != nil {
		t.Error("Fail to deserialize BlockVote")
	}

	if !reflect.DeepEqual(v1, v2) {
		t.Error("BlockVote serialization and deserialization don't match")
	}
}

package sharding

import (
	"testing"
)

func TestShard_ValidateShardID(t *testing.T) {
	tests := []struct {
		headers []*CollationHeader
	}{
		{
			headers: nil,
		}, {
			headers: nil,
		},
	}

	for _, tt := range tests {
		t.Logf("val: %v", tt.headers)
		if 0 == 1 {
			t.Fatalf("Wrong number of transactions. want=%d. got=%d", 5, 3)
		}
	}
}

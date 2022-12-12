package depositsnapshot

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
)

func hexString(t *testing.T, hexStr string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Error(err)
	}
	if len(b) != 32 {
		t.Fatalf("bad hash length, expected 32, got %d", len(b))
	}
	x := (*[32]byte)(b)
	return *x
}

func Test_create(t *testing.T) {
	tests := []struct {
		name   string
		leaves [][32]byte
		depth  uint64
		want   MerkleTreeNode
	}{
		{
			name:   "empty tree",
			leaves: nil,
			depth:  0,
			want:   &ZeroNode{},
		},
		{
			name:   "zero depth",
			leaves: [][32]byte{hexString(t, fmt.Sprintf("%064d", 0))},
			depth:  0,
			want:   &LeafNode{},
		},
		{
			name:   "depth of 1",
			leaves: [][32]byte{hexString(t, fmt.Sprintf("%064d", 0))},
			depth:  1,
			want:   &InnerNode{&LeafNode{}, &ZeroNode{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := create(tt.leaves, tt.depth); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("create() = %T, want %v", got, tt.want)
			}
		})
	}
}

func Test_fromSnapshotParts(t *testing.T) {
	tests := []struct {
		name      string
		finalized [][32]byte
		deposits  uint64
		level     uint64
		want      MerkleTreeNode
	}{
		{
			name:      "empty",
			finalized: nil,
			deposits:  0,
			level:     0,
			want:      &ZeroNode{},
		},
		{
			name:      "single finalized node",
			finalized: [][32]byte{hexString(t, fmt.Sprintf("%064d", 0))},
			deposits:  1,
			level:     0,
			want: &FinalizedNode{
				depositCount: 1,
				hash:         [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
		},
		//{
		//	name:      "multiple deposits and 1 finalized",
		//	finalized: [][32]byte{hexString(t, fmt.Sprintf("%064d", 0))},
		//	deposits:  2,
		//	level:     4,
		//	want: &InnerNode{
		//		left:  &InnerNode{&InnerNode{}, &ZeroNode{}},
		//		right: &ZeroNode{},
		//	},
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fromSnapshotParts(tt.finalized, tt.deposits, tt.level); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fromSnapshotParts() = %v, want %v", got, tt.want)
			}
		})
	}
}

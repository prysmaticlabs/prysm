package depositsnapshot

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
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
		{
			name:      "multiple deposits and 1 Finalized",
			finalized: [][32]byte{hexString(t, fmt.Sprintf("%064d", 0))},
			deposits:  2,
			level:     4,
			want: &InnerNode{
				left:  &InnerNode{&InnerNode{&FinalizedNode{depositCount: 2, hash: hexString(t, fmt.Sprintf("%064d", 0))}, &ZeroNode{1}}, &ZeroNode{2}},
				right: &ZeroNode{3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fromSnapshotParts(tt.finalized, tt.deposits, tt.level); !reflect.DeepEqual(got, tt.want) {
				tmp := got
				fmt.Println(tmp)
				t.Errorf("fromSnapshotParts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateProof(t *testing.T) {
	tests := []struct {
		name   string
		leaves uint64
	}{
		{
			name:   "1 leaf",
			leaves: 1,
		},
		{
			name:   "4 leaves",
			leaves: 4,
		},
		{
			name:   "10 leaves",
			leaves: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCases, err := readTestCases("test_cases.yaml")
			assert.NoError(t, err)
			tree := New()
			for _, c := range testCases[:tt.leaves] {
				err = tree.pushLeaf(c.DepositDataRoot)
				assert.NoError(t, err)
			}
			for i := uint64(0); i < tt.leaves; i++ {
				leaf, proof := generateProof(tree.tree, i, DepositContractDepth)
				assert.Equal(t, leaf, testCases[i].DepositDataRoot)
				calcRoot := merkleRootFromBranch(leaf, proof, i)
				assert.Equal(t, tree.tree.GetRoot(), calcRoot)
			}
		})
	}
}

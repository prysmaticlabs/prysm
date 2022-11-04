package depositsnapshot

import (
	"encoding/hex"
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

func Test_MerkleTreeNodeGetRoot(t *testing.T) {
	tests := []struct {
		name string
		node MerkleTreeNode
		want [32]byte
	}{
		{
			name: "FinalizedNode GetRoot",
			node: &FinalizedNode{
				deposits: 3,
				hash:     hexString(t, "7af7da533b0dc64b690cb0604f5a81e40ed83796dd14037ea3a55383b8f0976a"),
			},
			want: hexString(t, "7af7da533b0dc64b690cb0604f5a81e40ed83796dd14037ea3a55383b8f0976a"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := tt.node.GetRoot()
			assert.Equal(t, tt.want, hash)
		})
	}
}

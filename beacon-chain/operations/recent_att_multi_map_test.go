package operations

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
)

func TestRecentAttestationMultiMap_Contains(t *testing.T) {
	root := [32]byte{'F', 'O', 'O', 'B', 'A', 'R'}

	tests := []struct {
		inputs   []bitfield.Bitlist
		contains bitfield.Bitlist
		want     bool
	}{
		{
			inputs: []bitfield.Bitlist{
				{0b00000001, 0b1},
				{0b00000010, 0b1},
			},
			contains: bitfield.Bitlist{0b00000001, 0b1},
			want:     true,
		}, {
			inputs: []bitfield.Bitlist{
				{0b00111000, 0b1},
				{0b00000011, 0b1},
			},
			contains: bitfield.Bitlist{0b00000100, 0b1},
			want:     false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			mm := newRecentAttestationMultiMap()
			for _, input := range tt.inputs {
				mm.Insert(0, root, input)
			}
			if mm.Contains(root, tt.contains) != tt.want {
				t.Errorf("mm.Contains(root, tt.contains) = %v, wanted %v", mm.Contains(root, tt.contains), tt.want)
			}
		})
	}
}

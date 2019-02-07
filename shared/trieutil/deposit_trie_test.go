package trieutil

import (
	"testing"
)

func TestDepositTrie_UpdateDepositTrie(t *testing.T) {
	tests := []struct {
		deposits [][]byte
	}{
		{
			[][]byte{
				{1, 2, 3, 4},
				{5, 6, 7, 8},
			},
		},
		{
			[][]byte{
				{0, 0, 0, 0},
				{0, 0, 0, 0},
			},
		},
	}
	for _, tt := range tests {
		d := NewDepositTrie()
		d.UpdateDepositTrie(tt.deposits[0])
		if d.depositCount != 1 {
			t.Errorf("Expected deposit count to increase by 1, received %d", d.depositCount)
		}
		d.UpdateDepositTrie(tt.deposits[1])
		if d.depositCount != 2 {
			t.Errorf("Expected deposit count to increase by 1, received %d", d.depositCount)
		}

	}
}

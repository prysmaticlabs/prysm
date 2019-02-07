package trieutil

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
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

		hashedData := hashutil.Hash(tt.deposits[0])

		for i := 0; i < 32; i++ {
			hashedData = hashutil.Hash(append(hashedData[:], d.zeroHashes[i][:]...))
		}
		if d.Root() != hashedData {
			t.Errorf("Expected %#x but got %#x", hashedData, d.Root())
		}

		d.UpdateDepositTrie(tt.deposits[1])
		if d.depositCount != 2 {
			t.Errorf("Expected deposit count to increase by 1, received %d", d.depositCount)
		}

		hash1 := hashutil.Hash(tt.deposits[0])
		hash2 := hashutil.Hash(tt.deposits[1])

		hashedData = hashutil.Hash(append(hash1[:], hash2[:]...))

		for i := 0; i < 31; i++ {
			hashedData = hashutil.Hash(append(hashedData[:], d.zeroHashes[i+1][:]...))
		}
		if d.Root() != hashedData {
			t.Errorf("Expected %#x but got %#x", hashedData, d.Root())
		}

	}
}

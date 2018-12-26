package helperutils

import (
	"testing"
)

func TestMerkleRoot(t *testing.T) {
	valueSet := [][]byte{
		[]byte{'a'},
		[]byte{'b'},
		[]byte{'c'},
		[]byte{'d'},
	}

	t.Log(valueSet[3])
}

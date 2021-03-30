package stateutil

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// HandleByteArrays computes and returns byte arrays in a slice of root format.
func HandleByteArrays(val [][]byte, indices []uint64, convertAll bool) ([][32]byte, error) {
	length := len(indices)
	if convertAll {
		length = len(val)
	}
	roots := make([][32]byte, 0, length)
	rootCreator := func(input []byte) {
		newRoot := bytesutil.ToBytes32(input)
		roots = append(roots, newRoot)
	}
	if convertAll {
		for i := range val {
			rootCreator(val[i])
		}
		return roots, nil
	}
	if len(val) > 0 {
		for _, idx := range indices {
			if idx > uint64(len(val))-1 {
				return nil, fmt.Errorf("index %d greater than number of byte arrays %d", idx, len(val))
			}
			rootCreator(val[idx])
		}
	}
	return roots, nil
}

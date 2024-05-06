package stateutil

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
)

// SliceRoot computes the root of a slice of hashable objects.
func SliceRoot[T ssz.Hashable](slice []T, limit uint64) ([32]byte, error) {
	max := limit
	if uint64(len(slice)) > max {
		return [32]byte{}, fmt.Errorf("slice exceeds max length %d", max)
	}

	roots := make([][32]byte, len(slice))
	for i := 0; i < len(slice); i++ {
		r, err := slice[i].HashTreeRoot()
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not merkleize object")
		}
		roots[i] = r
	}

	sliceRoot, err := ssz.BitwiseMerkleize(roots, uint64(len(roots)), limit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not slice merkleization")
	}
	sliceLenBuf := new(bytes.Buffer)
	if err := binary.Write(sliceLenBuf, binary.LittleEndian, uint64(len(slice))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal slice length")
	}
	// We need to mix in the length of the slice.
	sliceLenRoot := make([]byte, 32)
	copy(sliceLenRoot, sliceLenBuf.Bytes())
	res := ssz.MixInLength(sliceRoot, sliceLenRoot)
	return res, nil
}

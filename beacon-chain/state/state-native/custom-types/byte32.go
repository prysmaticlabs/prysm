package customtypes

import (
	"fmt"

	fssz "github.com/ferranbt/fastssz"
)

var _ fssz.HashRoot = (Byte32)([32]byte{})
var _ fssz.Marshaler = (*Byte32)(nil)
var _ fssz.Unmarshaler = (*Byte32)(nil)

// Byte32 represents a 32 bytes Byte32 object in Ethereum beacon chain consensus.
type Byte32 [32]byte

// HashTreeRoot returns calculated hash root.
func (e Byte32) HashTreeRoot() ([32]byte, error) {
	return fssz.HashWithDefaultHasher(e)
}

// HashTreeRootWith hashes a Byte32 object with a Hasher from the default HasherPool.
func (e Byte32) HashTreeRootWith(hh *fssz.Hasher) error {
	hh.PutBytes(e[:])
	return nil
}

// UnmarshalSSZ deserializes the provided bytes buffer into the Byte32 object.
func (e *Byte32) UnmarshalSSZ(buf []byte) error {
	if len(buf) != e.SizeSSZ() {
		return fmt.Errorf("expected buffer of length %d received %d", e.SizeSSZ(), len(buf))
	}

	var b Byte32
	copy(b[:], buf)
	*e = b
	return nil
}

// MarshalSSZTo marshals Byte32 with the provided byte slice.
func (e *Byte32) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalled, err := e.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalled...), nil
}

// MarshalSSZ marshals Byte32 into a serialized object.
func (e *Byte32) MarshalSSZ() ([]byte, error) {
	return e[:], nil
}

// SizeSSZ returns the size of the serialized object.
func (_ *Byte32) SizeSSZ() int {
	return 32
}

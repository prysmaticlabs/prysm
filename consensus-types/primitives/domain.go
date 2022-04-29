package types

import (
	"fmt"

	fssz "github.com/ferranbt/fastssz"
)

var _ fssz.HashRoot = (Domain)([]byte{})
var _ fssz.Marshaler = (*Domain)(nil)
var _ fssz.Unmarshaler = (*Domain)(nil)

// Domain represents a 32 bytes domain object in Ethereum beacon chain consensus.
type Domain []byte

// HashTreeRoot --
func (e Domain) HashTreeRoot() ([32]byte, error) {
	return fssz.HashWithDefaultHasher(e)
}

// HashTreeRootWith --
func (e Domain) HashTreeRootWith(hh *fssz.Hasher) error {
	hh.PutBytes(e[:])
	return nil
}

// UnmarshalSSZ --
func (e *Domain) UnmarshalSSZ(buf []byte) error {
	if len(buf) != e.SizeSSZ() {
		return fmt.Errorf("expected buffer of length %d received %d", e.SizeSSZ(), len(buf))
	}

	var b [32]byte
	item := Domain(b[:])
	copy(item, buf)
	*e = item
	return nil
}

// MarshalSSZTo --
func (e *Domain) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalled, err := e.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalled...), nil
}

// MarshalSSZ --
func (e *Domain) MarshalSSZ() ([]byte, error) {
	return *e, nil
}

// SizeSSZ --
func (e *Domain) SizeSSZ() int {
	return 32
}

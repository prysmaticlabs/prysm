package sszutil

import (
	fssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
)

// SSZUint64 is a uint64 type that satisfies the fast-ssz interface.
type SSZUint64 uint64

// SizeSSZ returns the size of the serialized representation.
func (s *SSZUint64) SizeSSZ() int {
	return 8
}

// MarshalSSZTo marshals the uint64 with the provided byte slice.
func (s *SSZUint64) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalledObj, err := s.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalledObj...), nil
}

// MarshalSSZ Marshals the uint64 type into the serialized object.
func (s *SSZUint64) MarshalSSZ() ([]byte, error) {
	marshalledObj := fssz.MarshalUint64([]byte{}, uint64(*s))
	return marshalledObj, nil
}

// UnmarshalSSZ unmarshals the provided bytes buffer into the
// uint64 object.
func (s *SSZUint64) UnmarshalSSZ(buf []byte) error {
	if len(buf) != s.SizeSSZ() {
		return errors.Errorf("expected buffer with length of %d but received length %d", s.SizeSSZ(), len(buf))
	}
	*s = SSZUint64(fssz.UnmarshallUint64(buf))
	return nil
}

// HashTreeRoot hashes the uint64 object following the SSZ standard.
func (s *SSZUint64) HashTreeRoot() ([32]byte, error) {
	return htrutils.Uint64Root(uint64(*s)), nil
}

// HashTreeRootWith hashes the uint64 object with the given hasher.
func (s *SSZUint64) HashTreeRootWith(hh *fssz.Hasher) error {
	indx := hh.Index()
	hh.PutUint64(uint64(*s))
	hh.Merkleize(indx)
	return nil
}

// SSZUint64 is a bytes slice that satisfies the fast-ssz interface.
type SSZBytes []byte

// HashTreeRoot hashes the uint64 object following the SSZ standard.
func (b *SSZBytes) HashTreeRoot() ([32]byte, error) {
	return fssz.HashWithDefaultHasher(b)
}

// HashTreeRootWith hashes the uint64 object with the given hasher.
func (b *SSZBytes) HashTreeRootWith(hh *fssz.Hasher) error {
	indx := hh.Index()
	hh.PutBytes(*b)
	hh.Merkleize(indx)
	return nil
}

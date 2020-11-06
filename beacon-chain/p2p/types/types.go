// Package types contains all the respective p2p types that are required for sync
// but cannot be represented as a protobuf schema. This package also contains those
// types associated fast ssz methods.
package types

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// SSZUint64 is a uint64 type that satisfies the fast-ssz interface.
type SSZUint64 uint64

const rootLength = 32

const maxErrorLength = 256

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
	marshalledObj := ssz.MarshalUint64([]byte{}, uint64(*s))
	return marshalledObj, nil
}

// SizeSSZ returns the size of the serialized representation.
func (s *SSZUint64) SizeSSZ() int {
	return 8
}

// UnmarshalSSZ unmarshals the provided bytes buffer into the
// uint64 object.
func (s *SSZUint64) UnmarshalSSZ(buf []byte) error {
	if len(buf) != s.SizeSSZ() {
		return errors.Errorf("expected buffer with length of %d but received length %d", s.SizeSSZ(), len(buf))
	}
	*s = SSZUint64(ssz.UnmarshallUint64(buf))
	return nil
}

// BeaconBlockByRootsReq specifies the block by roots request type.
type BeaconBlockByRootsReq [][rootLength]byte

// MarshalSSZTo marshals the block by roots request with the provided byte slice.
func (s *BeaconBlockByRootsReq) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalledObj, err := s.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalledObj...), nil
}

// MarshalSSZ Marshals the block by roots request type into the serialized object.
func (s *BeaconBlockByRootsReq) MarshalSSZ() ([]byte, error) {
	if len(*s) > int(params.BeaconNetworkConfig().MaxRequestBlocks) {
		return nil, errors.Errorf("beacon block by roots request exceeds max size: %d > %d", len(*s), params.BeaconNetworkConfig().MaxRequestBlocks)
	}
	buf := make([]byte, 0, s.SizeSSZ())
	for _, r := range *s {
		buf = append(buf, r[:]...)
	}
	return buf, nil
}

// SizeSSZ returns the size of the serialized representation.
func (s *BeaconBlockByRootsReq) SizeSSZ() int {
	return len(*s) * rootLength
}

// UnmarshalSSZ unmarshals the provided bytes buffer into the
// block by roots request object.
func (s *BeaconBlockByRootsReq) UnmarshalSSZ(buf []byte) error {
	bufLen := len(buf)
	maxLength := int(params.BeaconNetworkConfig().MaxRequestBlocks * rootLength)
	if bufLen > maxLength {
		return errors.Errorf("expected buffer with length of upto %d but received length %d", maxLength, bufLen)
	}
	if bufLen%rootLength != 0 {
		return ssz.ErrIncorrectByteSize
	}
	numOfRoots := bufLen / rootLength
	roots := make([][rootLength]byte, 0, numOfRoots)
	for i := 0; i < numOfRoots; i++ {
		var rt [rootLength]byte
		copy(rt[:], buf[i*rootLength:(i+1)*rootLength])
		roots = append(roots, rt)
	}
	*s = roots
	return nil
}

// ErrorMessage describes the error message type.
type ErrorMessage []byte

// MarshalSSZTo marshals the error message with the provided byte slice.
func (s *ErrorMessage) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalledObj, err := s.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalledObj...), nil
}

// MarshalSSZ Marshals the error message into the serialized object.
func (s *ErrorMessage) MarshalSSZ() ([]byte, error) {
	if len(*s) > maxErrorLength {
		return nil, errors.Errorf("error message exceeds max size: %d > %d", len(*s), maxErrorLength)
	}
	buf := make([]byte, s.SizeSSZ())
	copy(buf, *s)
	return buf, nil
}

// SizeSSZ returns the size of the serialized representation.
func (s *ErrorMessage) SizeSSZ() int {
	return len(*s)
}

// UnmarshalSSZ unmarshals the provided bytes buffer into the
// error message object.
func (s *ErrorMessage) UnmarshalSSZ(buf []byte) error {
	bufLen := len(buf)
	maxLength := maxErrorLength
	if bufLen > maxLength {
		return errors.Errorf("expected buffer with length of upto %d but received length %d", maxLength, bufLen)
	}
	errMsg := make([]byte, bufLen)
	copy(errMsg, buf)
	*s = errMsg
	return nil
}

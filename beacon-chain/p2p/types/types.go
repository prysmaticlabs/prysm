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

func (s *SSZUint64) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalledObj, err := s.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalledObj...), nil
}

func (s *SSZUint64) MarshalSSZ() ([]byte, error) {
	marshalledObj := ssz.MarshalUint64([]byte{}, uint64(*s))
	return marshalledObj, nil
}

func (s *SSZUint64) SizeSSZ() int {
	return 8
}

func (s *SSZUint64) UnmarshalSSZ(buf []byte) error {
	if len(buf) != s.SizeSSZ() {
		return errors.Errorf("expected buffer with length of %d but received length %d", s.SizeSSZ(), len(buf))
	}
	*s = SSZUint64(ssz.UnmarshallUint64(buf))
	return nil
}

type BeaconBlockByRootsReq [][32]byte

func (s *BeaconBlockByRootsReq) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalledObj, err := s.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalledObj...), nil
}

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

func (s *BeaconBlockByRootsReq) SizeSSZ() int {
	return len(*s) * 32
}

func (s *BeaconBlockByRootsReq) UnmarshalSSZ(buf []byte) error {
	bufLen := len(buf)
	maxLength := int(params.BeaconNetworkConfig().MaxRequestBlocks * 32)
	if bufLen > maxLength {
		return errors.Errorf("expected buffer with length of upto %d but received length %d", maxLength, bufLen)
	}
	if bufLen%32 != 0 {
		return ssz.ErrIncorrectByteSize
	}
	numOfRoots := bufLen / 32
	roots := make([][32]byte, 0, numOfRoots)
	for i := 0; i < numOfRoots; i++ {
		var rt [32]byte
		copy(rt[:], buf[i*32:(i+1)*32])
		roots = append(roots, rt)
	}
	*s = roots
	return nil
}

type ErrorMessage []byte

func (s *ErrorMessage) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalledObj, err := s.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalledObj...), nil
}

func (s *ErrorMessage) MarshalSSZ() ([]byte, error) {
	if len(*s) > 256 {
		return nil, errors.Errorf("error message exceeds max size: %d > %d", len(*s), 256)
	}
	buf := make([]byte, s.SizeSSZ())
	copy(buf, *s)
	return buf, nil
}

func (s *ErrorMessage) SizeSSZ() int {
	return len(*s)
}

func (s *ErrorMessage) UnmarshalSSZ(buf []byte) error {
	bufLen := len(buf)
	maxLength := 256
	if bufLen > maxLength {
		return errors.Errorf("expected buffer with length of upto %d but received length %d", maxLength, bufLen)
	}
	errMsg := make([]byte, bufLen)
	copy(errMsg, buf)
	*s = errMsg
	return nil
}

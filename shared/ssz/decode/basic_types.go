package decode

import (
	"encoding/binary"
	"errors"

	uint128 "github.com/cockroachdb/cockroach/pkg/util/uint128"
)

// TODO work on error messages

func UnmarshalUint8(in byte) uint8 {
	return in
}

func UnmarshalUint16(in []byte) (uint16, error) {
	if len(in) != 2 {
		return uint16(0), errors.New("length other than 2 not allowed")
	}

	return binary.LittleEndian.Uint16(in), nil
}

func UnmarshalUint32(in []byte) (uint32, error) {
	if len(in) != 4 {
		return uint32(0), errors.New("length other than 4 not allowed")
	}

	return binary.LittleEndian.Uint32(in), nil
}

func UnmarshalUint64(in []byte) (uint64, error) {
	if len(in) != 8 {
		return uint64(0), errors.New("length other than 8 not allowed")
	}

	return binary.LittleEndian.Uint64(in), nil
}

func UnmarshalUint128(in []byte) (uint128.Uint128, error) { // TODO i might be unmarshaling it in the wrong order
	if len(in) != 16 {
		return uint128.Uint128{}, errors.New("length other than 16 not allowed")
	}

	hi := binary.LittleEndian.Uint64(in[:8])
	lo := binary.LittleEndian.Uint64(in[8:])

	return uint128.Uint128{hi, lo}, nil
}

func UnmarshalBoolean(in byte) (bool, error) { // TODO is this necessary?
	if in == 1 {
		return true, nil
	} else if in == 0 {
		return false, nil
	}

	return false, errors.New("input is neither 0 nor 1")
}

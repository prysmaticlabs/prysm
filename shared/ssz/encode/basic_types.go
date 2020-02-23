package encode

import "encoding/binary"

func MarshalUint8(in uint8) byte {
	return byte(in)
}

func MarshalUint16(in uint16) []byte {
	out := make([]byte, 2)

	binary.LittleEndian.PutUint16(out, in)

	return out
}

func MarshalUint32(in uint32) []byte {
	out := make([]byte, 4)

	binary.LittleEndian.PutUint32(out, in)

	return out
}

func MarshalUint64(in uint64) []byte {
	out := make([]byte, 8)

	binary.LittleEndian.PutUint64(out, in)

	return out
}

func MarshalBoolean(in bool) byte { // TODO is this necessary?
	if in {
		return 1
	}

	return 0
}

package stateutil

import "encoding/binary"

// HashFn describes a hash function and its associated bytes buffer
type HashFn struct {
	f           func(input []byte) [32]byte
	bytesBuffer [64]byte
}

// Combi describes a method which merges two 32-byte arrays and hashes
// them.
func (h HashFn) Combi(a [32]byte, b [32]byte) [32]byte {
	copy(h.bytesBuffer[:32], a[:])
	copy(h.bytesBuffer[32:], b[:])
	return h.f(h.bytesBuffer[:])
}

// MixIn describes a method where we add in the provided
// integer to the end of the byte array and hash it.
func (h HashFn) MixIn(a [32]byte, i uint64) [32]byte {
	copy(h.bytesBuffer[:32], a[:])
	copy(h.bytesBuffer[32:], make([]byte, 32, 32))
	binary.LittleEndian.PutUint64(h.bytesBuffer[32:], i)
	return h.f(h.bytesBuffer[:])
}

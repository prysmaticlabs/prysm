package stateutil

import "encoding/binary"

type HashFn func(input []byte) [32]byte

// Hasher describes an interface through which we can
// perform hash operations on byte arrays,indices,etc.
type Hasher interface {
	Hash(a []byte) [32]byte
	Combi(a [32]byte, b [32]byte) [32]byte
	MixIn(a [32]byte, i uint64) [32]byte
}

type HasherFunc struct {
	b        [64]byte
	hashFunc HashFn
}

// NewHasherFunc is the constructor for the object
// that fulfills the Hasher interface.
func NewHasherFunc(h HashFn) *HasherFunc {
	return &HasherFunc{
		b:        [64]byte{},
		hashFunc: h,
	}
}

// Hash utilizes the provided hash function for
// the object.
func (h *HasherFunc) Hash(a []byte) [32]byte {
	return h.hashFunc(a)
}

func (h *HasherFunc) Combi(a [32]byte, b [32]byte) [32]byte {
	copy(h.b[:32], a[:])
	copy(h.b[32:], b[:])
	return h.Hash(h.b[:])
}

func (h *HasherFunc) MixIn(a [32]byte, i uint64) [32]byte {
	copy(h.b[:32], a[:])
	copy(h.b[32:], make([]byte, 32, 32))
	binary.LittleEndian.PutUint64(h.b[32:], i)
	return h.Hash(h.b[:])
}

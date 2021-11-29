// +build minimal

package customtypes

import (
	"fmt"

	fssz "github.com/ferranbt/fastssz"
)

const (
	// Number of block roots in the beacon state.
	BlockRootsSize = 64
	// Number of state roots in the beacon state.
	StateRootsSize = 64
	// Number of randao mixes in the beacon state,
	RandaoMixesSize = 64
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
func (e *Byte32) SizeSSZ() int {
	return 32
}

var _ fssz.HashRoot = (StateRoots)([BlockRootsSize][32]byte{})
var _ fssz.Marshaler = (*StateRoots)(nil)
var _ fssz.Unmarshaler = (*StateRoots)(nil)

// Byte32 represents a 32 bytes StateRoots object in Ethereum beacon chain consensus.
type StateRoots [BlockRootsSize][32]byte

// HashTreeRoot returns calculated hash root.
func (r StateRoots) HashTreeRoot() ([32]byte, error) {
	return fssz.HashWithDefaultHasher(r)
}

// HashTreeRootWith hashes a StateRoots object with a Hasher from the default HasherPool.
func (r StateRoots) HashTreeRootWith(hh *fssz.Hasher) error {
	index := hh.Index()
	for _, sRoot := range r {
		hh.Append(sRoot[:])
	}
	hh.Merkleize(index)
	return nil
}

// UnmarshalSSZ deserializes the provided bytes buffer into the StateRoots object.
func (r *StateRoots) UnmarshalSSZ(buf []byte) error {
	if len(buf) != r.SizeSSZ() {
		return fmt.Errorf("expected buffer of length %d received %d", r.SizeSSZ(), len(buf))
	}

	var roots StateRoots
	for i, _ := range roots {
		copy(roots[i][:], buf[i*32:(i+1)*32])
	}
	*r = roots
	return nil
}

// MarshalSSZTo marshals StateRoots with the provided byte slice.
func (r *StateRoots) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalled, err := r.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalled...), nil
}

// MarshalSSZ marshals StateRoots into a serialized object.
func (r *StateRoots) MarshalSSZ() ([]byte, error) {
	marshalled := make([]byte, BlockRootsSize*32)
	for i, r32 := range r {
		for j, rr := range r32 {
			marshalled[i*32+j] = rr
		}
	}
	return marshalled, nil
}

// SizeSSZ returns the size of the serialized object.
func (r *StateRoots) SizeSSZ() int {
	return BlockRootsSize * 32
}

var _ fssz.HashRoot = (RandaoMixes)([RandaoMixesSize][32]byte{})
var _ fssz.Marshaler = (*RandaoMixes)(nil)
var _ fssz.Unmarshaler = (*RandaoMixes)(nil)

// Byte32 represents a 32 bytes RandaoMixes object in Ethereum beacon chain consensus.
type RandaoMixes [RandaoMixesSize][32]byte

// HashTreeRoot returns calculated hash root.
func (r RandaoMixes) HashTreeRoot() ([32]byte, error) {
	return fssz.HashWithDefaultHasher(r)
}

// HashTreeRootWith hashes a RandaoMixes object with a Hasher from the default HasherPool.
func (r RandaoMixes) HashTreeRootWith(hh *fssz.Hasher) error {
	index := hh.Index()
	for _, sRoot := range r {
		hh.Append(sRoot[:])
	}
	hh.Merkleize(index)
	return nil
}

// UnmarshalSSZ deserializes the provided bytes buffer into the RandaoMixes object.
func (r *RandaoMixes) UnmarshalSSZ(buf []byte) error {
	if len(buf) != r.SizeSSZ() {
		return fmt.Errorf("expected buffer of length %d received %d", r.SizeSSZ(), len(buf))
	}

	var roots RandaoMixes
	for i, _ := range roots {
		copy(roots[i][:], buf[i*32:(i+1)*32])
	}
	*r = roots
	return nil
}

// MarshalSSZTo marshals RandaoMixes with the provided byte slice.
func (r *RandaoMixes) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalled, err := r.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalled...), nil
}

// MarshalSSZ marshals RandaoMixes into a serialized object.
func (r *RandaoMixes) MarshalSSZ() ([]byte, error) {
	marshalled := make([]byte, RandaoMixesSize*32)
	for i, r32 := range r {
		for j, rr := range r32 {
			marshalled[i*32+j] = rr
		}
	}
	return marshalled, nil
}

// SizeSSZ returns the size of the serialized object.
func (r *RandaoMixes) SizeSSZ() int {
	return RandaoMixesSize * 32
}

var _ fssz.HashRoot = (HistoricalRoots)([][32]byte{})
var _ fssz.Marshaler = (*HistoricalRoots)(nil)
var _ fssz.Unmarshaler = (*HistoricalRoots)(nil)

// Byte32 represents a 32 bytes HistoricalRoots object in Ethereum beacon chain consensus.
type HistoricalRoots [][32]byte

// HashTreeRoot returns calculated hash root.
func (r HistoricalRoots) HashTreeRoot() ([32]byte, error) {
	return fssz.HashWithDefaultHasher(r)
}

// HashTreeRootWith hashes a HistoricalRoots object with a Hasher from the default HasherPool.
func (r HistoricalRoots) HashTreeRootWith(hh *fssz.Hasher) error {
	index := hh.Index()
	for _, sRoot := range r {
		hh.Append(sRoot[:])
	}
	hh.Merkleize(index)
	return nil
}

// UnmarshalSSZ deserializes the provided bytes buffer into the HistoricalRoots object.
func (r *HistoricalRoots) UnmarshalSSZ(buf []byte) error {
	if len(buf) != r.SizeSSZ() {
		return fmt.Errorf("expected buffer of length %d received %d", r.SizeSSZ(), len(buf))
	}

	mixes := make([][32]byte, len(buf)/32)
	for i, _ := range mixes {
		copy(mixes[i][:], buf[i*32:(i+1)*32])
	}
	*r = mixes
	return nil
}

// MarshalSSZTo marshals HistoricalRoots with the provided byte slice.
func (r *HistoricalRoots) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalled, err := r.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalled...), nil
}

// MarshalSSZ marshals HistoricalRoots into a serialized object.
func (r *HistoricalRoots) MarshalSSZ() ([]byte, error) {
	marshalled := make([]byte, len(*r)*32)
	for i, r32 := range *r {
		for j, rr := range r32 {
			marshalled[i*32+j] = rr
		}
	}
	return marshalled, nil
}

// SizeSSZ returns the size of the serialized object.
func (r *HistoricalRoots) SizeSSZ() int {
	return len(*r) * 32
}

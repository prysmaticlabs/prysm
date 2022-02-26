package customtypes

import (
	"fmt"
	"sync"

	fssz "github.com/ferranbt/fastssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
)

var _ fssz.HashRoot = (*BlockRoots)(nil)
var _ fssz.Marshaler = (*BlockRoots)(nil)
var _ fssz.Unmarshaler = (*BlockRoots)(nil)

type Indexer interface {
	RootAtIndex(idx uint64) [32]byte
	TotalLength() uint64
}

// BlockRoots represents block roots of the beacon state.
type BlockRoots struct {
	baseArray    *baseArray
	fieldJournal map[uint64][32]byte
	*stateutil.Reference
}

type baseArray struct {
	baseArray *[fieldparams.BlockRootsLength][32]byte
	*sync.RWMutex
	*stateutil.Reference
}

func (b *baseArray) RootAtIndex(idx uint64) [32]byte {
	b.RWMutex.RLock()
	defer b.RWMutex.RUnlock()
	return b.baseArray[idx]
}

func (b *baseArray) TotalLength() uint64 {
	return fieldparams.BlockRootsLength
}

// HashTreeRoot returns calculated hash root.
func (r *BlockRoots) HashTreeRoot() ([32]byte, error) {
	return fssz.HashWithDefaultHasher(r)
}

// HashTreeRootWith hashes a BlockRoots object with a Hasher from the default HasherPool.
func (r *BlockRoots) HashTreeRootWith(hh *fssz.Hasher) error {
	index := hh.Index()

	for i := uint64(0); i < r.baseArray.TotalLength(); i++ {
		if val, ok := r.fieldJournal[i]; ok {
			hh.Append(val[:])
			continue
		}
		rt := r.baseArray.RootAtIndex(i)
		hh.Append(rt[:])
	}
	hh.Merkleize(index)
	return nil
}

// UnmarshalSSZ deserializes the provided bytes buffer into the BlockRoots object.
func (r *BlockRoots) UnmarshalSSZ(buf []byte) error {
	if len(buf) != r.SizeSSZ() {
		return fmt.Errorf("expected buffer of length %d received %d", r.SizeSSZ(), len(buf))
	}
	r.baseArray.Lock()
	defer r.baseArray.Unlock()

	for i := range r.baseArray.baseArray {
		copy(r.baseArray.baseArray[i][:], buf[i*32:(i+1)*32])
	}

	return nil
}

// MarshalSSZTo marshals BlockRoots with the provided byte slice.
func (r *BlockRoots) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalled, err := r.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalled...), nil
}

// MarshalSSZ marshals BlockRoots into a serialized object.
func (r *BlockRoots) MarshalSSZ() ([]byte, error) {
	marshalled := make([]byte, fieldparams.BlockRootsLength*32)
	for i := uint64(0); i < r.baseArray.TotalLength(); i++ {
		if val, ok := r.fieldJournal[i]; ok {
			copy(marshalled[i*32:], val[:])
			continue
		}
		rt := r.baseArray.RootAtIndex(i)
		copy(marshalled[i*32:], rt[:])
	}
	return marshalled, nil
}

// SizeSSZ returns the size of the serialized object.
func (_ *BlockRoots) SizeSSZ() int {
	return fieldparams.BlockRootsLength * 32
}

// Slice converts a customtypes.BlockRoots object into a 2D byte slice.
func (r *BlockRoots) Slice() [][]byte {
	if r == nil {
		return nil
	}
	bRoots := make([][]byte, r.baseArray.TotalLength())
	for i := uint64(0); i < r.baseArray.TotalLength(); i++ {
		if val, ok := r.fieldJournal[i]; ok {
			bRoots[i] = val[:]
			continue
		}
		rt := r.baseArray.RootAtIndex(i)
		bRoots[i] = rt[:]
	}
	return bRoots
}

func SetFromSlice(slice [][]byte) *BlockRoots {
	br := &BlockRoots{
		baseArray: &baseArray{
			baseArray: new([fieldparams.BlockRootsLength][32]byte),
			RWMutex:   new(sync.RWMutex),
			Reference: stateutil.NewRef(1),
		},
		fieldJournal: map[uint64][32]byte{},
		Reference:    stateutil.NewRef(1),
	}
	for i, rt := range slice {
		copy(br.baseArray.baseArray[i][:], rt)
	}
	return br
}

func (r *BlockRoots) SetFromBaseField(field [fieldparams.BlockRootsLength][32]byte) {
	r.baseArray.baseArray = &field
}

func (r *BlockRoots) RootAtIndex(idx uint64) [32]byte {
	if val, ok := r.fieldJournal[idx]; ok {
		return val
	}
	return r.baseArray.RootAtIndex(idx)
}

func (r *BlockRoots) SetRootAtIndex(idx uint64, val [32]byte) {
	if r.Refs() <= 1 && r.baseArray.Refs() <= 1 {
		r.baseArray.baseArray[idx] = val
		return
	}
	if r.Refs() <= 1 {
		r.fieldJournal[idx] = val
		return
	}
	newJournal := make(map[uint64][32]byte)
	for k, val := range r.fieldJournal {
		newJournal[k] = val
	}
	r.fieldJournal = newJournal
	r.MinusRef()
	r.Reference = stateutil.NewRef(1)
	r.fieldJournal[idx] = val
}

func (r *BlockRoots) TotalLength() uint64 {
	return fieldparams.BlockRootsLength
}

func (r *BlockRoots) IncreaseRef() {
	r.Reference.AddRef()
	r.baseArray.Reference.AddRef()
}

func (r *BlockRoots) DecreaseRef() {
	r.Reference.MinusRef()
	r.baseArray.Reference.MinusRef()
}

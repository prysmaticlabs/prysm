package customtypes

import (
	"fmt"
	"sync"

	fssz "github.com/ferranbt/fastssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
)

var _ fssz.HashRoot = (*RandaoMixes)(nil)
var _ fssz.Marshaler = (*RandaoMixes)(nil)
var _ fssz.Unmarshaler = (*RandaoMixes)(nil)

// BlockRoots represents block roots of the beacon state.
type RandaoMixes struct {
	baseArray    *baseArrayRandaoMixes
	fieldJournal map[uint64][32]byte
	*stateutil.Reference
}

type baseArrayRandaoMixes struct {
	baseArray *[fieldparams.RandaoMixesLength][32]byte
	*sync.RWMutex
	*stateutil.Reference
}

func (b *baseArrayRandaoMixes) RootAtIndex(idx uint64) [32]byte {
	b.RWMutex.RLock()
	defer b.RWMutex.RUnlock()
	return b.baseArray[idx]
}

func (b *baseArrayRandaoMixes) TotalLength() uint64 {
	return fieldparams.RandaoMixesLength
}

// HashTreeRoot returns calculated hash root.
func (r *RandaoMixes) HashTreeRoot() ([32]byte, error) {
	return fssz.HashWithDefaultHasher(r)
}

// HashTreeRootWith hashes a BlockRoots object with a Hasher from the default HasherPool.
func (r *RandaoMixes) HashTreeRootWith(hh *fssz.Hasher) error {
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
func (r *RandaoMixes) UnmarshalSSZ(buf []byte) error {
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
func (r *RandaoMixes) MarshalSSZTo(dst []byte) ([]byte, error) {
	marshalled, err := r.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(dst, marshalled...), nil
}

// MarshalSSZ marshals BlockRoots into a serialized object.
func (r *RandaoMixes) MarshalSSZ() ([]byte, error) {
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
func (_ *RandaoMixes) SizeSSZ() int {
	return fieldparams.RandaoMixesLength * 32
}

// Slice converts a customtypes.BlockRoots object into a 2D byte slice.
func (r *RandaoMixes) Slice() [][]byte {
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

func SetFromSliceRandao(slice [][]byte) *RandaoMixes {
	br := &RandaoMixes{
		baseArray: &baseArrayRandaoMixes{
			baseArray: new([fieldparams.RandaoMixesLength][32]byte),
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

func (r *RandaoMixes) SetFromBaseField(field [fieldparams.RandaoMixesLength][32]byte) {
	r.baseArray.baseArray = &field
}

func (r *RandaoMixes) RootAtIndex(idx uint64) [32]byte {
	if val, ok := r.fieldJournal[idx]; ok {
		return val
	}
	return r.baseArray.RootAtIndex(idx)
}

func (r *RandaoMixes) SetRootAtIndex(idx uint64, val [32]byte) {
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

func (r *RandaoMixes) TotalLength() uint64 {
	return fieldparams.RandaoMixesLength
}

func (r *RandaoMixes) IncreaseRef() {
	r.Reference.AddRef()
	r.baseArray.Reference.AddRef()
}

func (r *RandaoMixes) DecreaseRef() {
	r.Reference.MinusRef()
	r.baseArray.Reference.MinusRef()
}

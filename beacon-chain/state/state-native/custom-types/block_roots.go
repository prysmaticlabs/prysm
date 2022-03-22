package customtypes

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"unsafe"

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
	baseArray    *baseArrayBlockRoots
	fieldJournal map[uint64][32]byte
	generation   uint64
	*stateutil.Reference
}

type baseArrayBlockRoots struct {
	baseArray     *[fieldparams.BlockRootsLength][32]byte
	descendantMap map[uint64][]uintptr
	*sync.RWMutex
	*stateutil.Reference
}

type sorter struct {
	objs        [][]uintptr
	generations []uint64
}

func (s sorter) Len() int {
	return len(s.generations)
}

func (s sorter) Swap(i, j int) {
	s.objs[i], s.objs[j] = s.objs[j], s.objs[i]
	s.generations[i], s.generations[j] = s.generations[j], s.generations[i]
}

func (s sorter) Less(i, j int) bool {
	return s.generations[i] < s.generations[j]
}

func (b *baseArrayBlockRoots) RootAtIndex(idx uint64) [32]byte {
	b.RWMutex.RLock()
	defer b.RWMutex.RUnlock()
	return b.baseArray[idx]
}

func (b *baseArrayBlockRoots) TotalLength() uint64 {
	return fieldparams.BlockRootsLength
}

func (b *baseArrayBlockRoots) addGeneration(generation uint64, descendant uintptr) {
	b.RWMutex.Lock()
	defer b.RWMutex.Unlock()
	b.descendantMap[generation] = append(b.descendantMap[generation], descendant)
}

func (b *baseArrayBlockRoots) removeGeneration(generation uint64, descendant uintptr) {
	b.RWMutex.Lock()
	defer b.RWMutex.Unlock()
	ptrVals := b.descendantMap[generation]
	newVals := []uintptr{}
	for _, v := range ptrVals {
		if v == descendant {
			continue
		}
		newVals = append(newVals, v)
	}
	b.descendantMap[generation] = newVals
}

func (b *baseArrayBlockRoots) numOfDescendants() uint64 {
	b.RWMutex.RLock()
	defer b.RWMutex.RUnlock()
	return uint64(len(b.descendantMap))
}

func (b *baseArrayBlockRoots) cleanUp() {
	b.RWMutex.Lock()
	defer b.RWMutex.Unlock()
	fmt.Printf("\n cleaning up block roots %d \n ", len(b.descendantMap))
	listOfObjs := [][]uintptr{}
	generations := []uint64{}
	for g, objs := range b.descendantMap {
		generations = append(generations, g)
		listOfObjs = append(listOfObjs, objs)
	}
	sortedObj := sorter{
		objs:        listOfObjs,
		generations: generations,
	}
	sort.Sort(sortedObj)
	lastReferencedGen := 0
	lastRefrencedIdx := 0
	lastRefPointer := 0
	for i, g := range sortedObj.generations {
		for j, o := range sortedObj.objs[i] {

			x := (*BlockRoots)(unsafe.Pointer(o))
			if x == nil {
				continue
			}

			lastReferencedGen = int(g) // lint:ignore uintcast- ajhdjhd
			lastRefrencedIdx = i
			lastRefPointer = j
			break
		}
		if lastReferencedGen != 0 {
			break
		}
	}
	fmt.Printf("\n block root map %d, %d, %d \n ", lastReferencedGen, lastRefrencedIdx, lastRefPointer)

	br := (*BlockRoots)(unsafe.Pointer(sortedObj.objs[lastRefrencedIdx][lastRefPointer]))
	for k, v := range br.fieldJournal {
		b.baseArray[k] = v
	}
	sortedObj.generations = sortedObj.generations[lastRefrencedIdx:]
	sortedObj.objs = sortedObj.objs[lastRefrencedIdx:]

	newMap := make(map[uint64][]uintptr)
	for i, g := range sortedObj.generations {
		newMap[g] = sortedObj.objs[i]
	}
	b.descendantMap = newMap
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

// Slice converts a customtypes.BlockRoots object into a 2D byte slice.
func (r *BlockRoots) Array() [fieldparams.BlockRootsLength][32]byte {
	if r == nil {
		return [fieldparams.BlockRootsLength][32]byte{}
	}
	bRoots := [fieldparams.BlockRootsLength][32]byte{}
	for i := uint64(0); i < r.baseArray.TotalLength(); i++ {
		if val, ok := r.fieldJournal[i]; ok {
			bRoots[i] = val
			continue
		}
		rt := r.baseArray.RootAtIndex(i)
		bRoots[i] = rt
	}
	return bRoots
}

func SetFromSlice(slice [][]byte) *BlockRoots {
	br := &BlockRoots{
		baseArray: &baseArrayBlockRoots{
			baseArray:     new([fieldparams.BlockRootsLength][32]byte),
			descendantMap: map[uint64][]uintptr{},
			RWMutex:       new(sync.RWMutex),
			Reference:     stateutil.NewRef(1),
		},
		fieldJournal: map[uint64][32]byte{},
		Reference:    stateutil.NewRef(1),
	}
	for i, rt := range slice {
		copy(br.baseArray.baseArray[i][:], rt)
	}
	runtime.SetFinalizer(br, blockRootsFinalizer)
	return br
}

func (r *BlockRoots) SetFromBaseField(field [fieldparams.BlockRootsLength][32]byte) {
	r.baseArray = &baseArrayBlockRoots{
		baseArray:     &field,
		descendantMap: map[uint64][]uintptr{},
		RWMutex:       new(sync.RWMutex),
		Reference:     stateutil.NewRef(1),
	}
	r.fieldJournal = map[uint64][32]byte{}
	r.Reference = stateutil.NewRef(1)
	r.baseArray.addGeneration(0, reflect.ValueOf(r).Pointer())
	runtime.SetFinalizer(r, blockRootsFinalizer)
}

func (r *BlockRoots) RootAtIndex(idx uint64) [32]byte {
	if val, ok := r.fieldJournal[idx]; ok {
		return val
	}
	return r.baseArray.RootAtIndex(idx)
}

func (r *BlockRoots) SetRootAtIndex(idx uint64, val [32]byte) {
	if r.Refs() <= 1 && r.baseArray.Refs() <= 1 {
		r.baseArray.Lock()
		r.baseArray.baseArray[idx] = val
		r.baseArray.Unlock()
		return
	}
	if r.Refs() <= 1 {
		r.fieldJournal[idx] = val
		r.baseArray.removeGeneration(r.generation, reflect.ValueOf(r).Pointer())
		r.generation++
		r.baseArray.addGeneration(r.generation, reflect.ValueOf(r).Pointer())
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
	r.baseArray.removeGeneration(r.generation, reflect.ValueOf(r).Pointer())
	r.generation++
	r.baseArray.addGeneration(r.generation, reflect.ValueOf(r).Pointer())
}

func (r *BlockRoots) Copy() *BlockRoots {
	r.baseArray.AddRef()
	r.Reference.AddRef()
	br := &BlockRoots{
		baseArray:    r.baseArray,
		fieldJournal: r.fieldJournal,
		Reference:    r.Reference,
		generation:   r.generation,
	}
	r.baseArray.addGeneration(r.generation, reflect.ValueOf(br).Pointer())
	if r.baseArray.numOfDescendants() > 20 {
		r.baseArray.cleanUp()
	}
	runtime.SetFinalizer(br, blockRootsFinalizer)
	return br
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

func blockRootsFinalizer(br *BlockRoots) {
	br.baseArray.Lock()
	defer br.baseArray.Unlock()
	ptrVal := reflect.ValueOf(br).Pointer()
	vals, ok := br.baseArray.descendantMap[br.generation]
	if !ok {
		return
	}
	exists := false
	wantedVals := []uintptr{}
	for _, v := range vals {
		if v == ptrVal {
			exists = true
			continue
		}
		newV := v
		wantedVals = append(wantedVals, newV)
	}
	if !exists {
		return
	}
	br.baseArray.descendantMap[br.generation] = wantedVals
}

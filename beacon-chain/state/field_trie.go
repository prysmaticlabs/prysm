package state

import (
	"sync"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/memorypool"
)

type FieldTrie struct {
	*sync.Mutex
	*reference
	fieldLayers [][]*[32]byte
	field       fieldIndex
}

func NewFieldTrie(field fieldIndex, elements [][]byte, length uint64) *FieldTrie {
	return &FieldTrie{
		fieldLayers: stateutil.ReturnTrieLayer(elements, length),
		field:       field,
		reference:   &reference{1},
		Mutex:       new(sync.Mutex),
	}
}

func (f *FieldTrie) RecomputeTrie(indices []uint64, elements [][]byte) ([32]byte, error) {
	f.Lock()
	defer f.Unlock()
	var err error
	var fieldRoot [32]byte
	for _, idx := range indices {
		root := bytesutil.ToBytes32(elements[idx])
		f.fieldLayers[0][idx] = &root
	}
	fieldRoot, f.fieldLayers, err = stateutil.RecomputeFromLayer(f.fieldLayers, indices)
	if err != nil {
		return [32]byte{}, err
	}
	return fieldRoot, nil
}

func (f *FieldTrie) CopyTrie() *FieldTrie {
	//f.Mutex.Lock()
	//defer f.Mutex.Unlock()
	if f.fieldLayers == nil {
		return &FieldTrie{
			field:     f.field,
			reference: &reference{1},
			Mutex:     new(sync.Mutex),
		}
	}
	dstFieldTrie := [][]*[32]byte{}
	switch f.field {
	case randaoMixes:
		dstFieldTrie = memorypool.GetTripleByteSliceRandaoMixes(len(f.fieldLayers))
	case blockRoots:
		dstFieldTrie = memorypool.GetTripleByteSliceBlockRoots(len(f.fieldLayers))
	case stateRoots:
		dstFieldTrie = memorypool.GetTripleByteSliceStateRoots(len(f.fieldLayers))
	default:
		dstFieldTrie = make([][]*[32]byte, len(f.fieldLayers))
	}

	for i, layer := range f.fieldLayers {
		if len(dstFieldTrie[i]) < len(layer) {
			diffSlice := make([]*[32]byte, len(layer)-len(dstFieldTrie[i]))
			dstFieldTrie[i] = append(dstFieldTrie[i], diffSlice...)
		}
		dstFieldTrie[i] = dstFieldTrie[i][:len(layer)]
		copy(dstFieldTrie[i], layer)
	}
	return &FieldTrie{
		fieldLayers: dstFieldTrie,
		field:       f.field,
		reference:   &reference{1},
		Mutex:       new(sync.Mutex),
	}
}

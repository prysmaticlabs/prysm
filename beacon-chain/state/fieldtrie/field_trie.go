package fieldtrie

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
)

// FieldTrie is the representation of the representative
// trie of the particular field.
type FieldTrie struct {
	*sync.RWMutex
	Reference   *stateutil.Reference
	FieldLayers [][]*[32]byte
	field       types.FieldIndex
	datType     types.DataType
	length      uint64
}

// NewFieldTrie is the constructor for the field trie data structure. It creates the corresponding
// trie according to the given parameters. Depending on whether the field is a basic/composite array
// which is either fixed/variable length, it will appropriately determine the trie.
func NewFieldTrie(field types.FieldIndex, dataType types.DataType, elements interface{}, length uint64) (*FieldTrie, error) {
	if elements == nil {
		return &FieldTrie{
			field:     field,
			datType:   dataType,
			Reference: stateutil.NewRef(1),
			RWMutex:   new(sync.RWMutex),
			length:    length,
		}, nil
	}
	fieldRoots, err := fieldConverters(field, []uint64{}, elements, true)
	if err != nil {
		return nil, err
	}
	if err := validateElements(field, elements, length); err != nil {
		return nil, err
	}
	switch dataType {
	case types.BasicArray:
		return &FieldTrie{
			FieldLayers: stateutil.ReturnTrieLayer(fieldRoots, length),
			field:       field,
			datType:     dataType,
			Reference:   stateutil.NewRef(1),
			RWMutex:     new(sync.RWMutex),
			length:      length,
		}, nil
	case types.CompositeArray:
		return &FieldTrie{
			FieldLayers: stateutil.ReturnTrieLayerVariable(fieldRoots, length),
			field:       field,
			datType:     dataType,
			Reference:   stateutil.NewRef(1),
			RWMutex:     new(sync.RWMutex),
			length:      length,
		}, nil
	default:
		return nil, errors.Errorf("unrecognized data type in field map: %v", reflect.TypeOf(dataType).Name())
	}

}

// RecomputeTrie rebuilds the affected branches in the trie according to the provided
// changed indices and elements. This recomputes the trie according to the particular
// field the trie is based on.
func (f *FieldTrie) RecomputeTrie(indices []uint64, elements interface{}) ([32]byte, error) {
	f.Lock()
	defer f.Unlock()
	var fieldRoot [32]byte
	if len(indices) == 0 {
		return f.TrieRoot()
	}
	fieldRoots, err := fieldConverters(f.field, indices, elements, false)
	if err != nil {
		return [32]byte{}, err
	}
	if err := f.validateIndices(indices); err != nil {
		return [32]byte{}, err
	}
	switch f.datType {
	case types.BasicArray:
		fieldRoot, f.FieldLayers, err = stateutil.RecomputeFromLayer(fieldRoots, indices, f.FieldLayers)
		if err != nil {
			return [32]byte{}, err
		}
		return fieldRoot, nil
	case types.CompositeArray:
		fieldRoot, f.FieldLayers, err = stateutil.RecomputeFromLayerVariable(fieldRoots, indices, f.FieldLayers)
		if err != nil {
			return [32]byte{}, err
		}
		return stateutil.AddInMixin(fieldRoot, uint64(len(f.FieldLayers[0])))
	default:
		return [32]byte{}, errors.Errorf("unrecognized data type in field map: %v", reflect.TypeOf(f.datType).Name())
	}
}

// CopyTrie copies the references to the elements the trie
// is built on.
func (f *FieldTrie) CopyTrie() *FieldTrie {
	if f.FieldLayers == nil {
		return &FieldTrie{
			field:     f.field,
			datType:   f.datType,
			Reference: stateutil.NewRef(1),
			RWMutex:   new(sync.RWMutex),
			length:    f.length,
		}
	}
	dstFieldTrie := make([][]*[32]byte, len(f.FieldLayers))
	for i, layer := range f.FieldLayers {
		dstFieldTrie[i] = make([]*[32]byte, len(layer))
		copy(dstFieldTrie[i], layer)
	}
	return &FieldTrie{
		FieldLayers: dstFieldTrie,
		field:       f.field,
		datType:     f.datType,
		Reference:   stateutil.NewRef(1),
		RWMutex:     new(sync.RWMutex),
		length:      f.length,
	}
}

// TrieRoot returns the corresponding root of the trie.
func (f *FieldTrie) TrieRoot() ([32]byte, error) {
	switch f.datType {
	case types.BasicArray:
		return *f.FieldLayers[len(f.FieldLayers)-1][0], nil
	case types.CompositeArray:
		trieRoot := *f.FieldLayers[len(f.FieldLayers)-1][0]
		return stateutil.AddInMixin(trieRoot, uint64(len(f.FieldLayers[0])))
	default:
		return [32]byte{}, errors.Errorf("unrecognized data type in field map: %v", reflect.TypeOf(f.datType).Name())
	}
}

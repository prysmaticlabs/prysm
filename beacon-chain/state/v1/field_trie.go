package v1

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
)

// FieldTrie is the representation of the representative
// trie of the particular field.
type FieldTrie struct {
	*sync.RWMutex
	reference   *stateutil.Reference
	fieldLayers [][]*[32]byte
	field       fieldIndex
	length      uint64
}

// NewFieldTrie is the constructor for the field trie data structure. It creates the corresponding
// trie according to the given parameters. Depending on whether the field is a basic/composite array
// which is either fixed/variable length, it will appropriately determine the trie.
func NewFieldTrie(field fieldIndex, elements interface{}, length uint64) (*FieldTrie, error) {
	if elements == nil {
		return &FieldTrie{
			field:     field,
			reference: stateutil.NewRef(1),
			RWMutex:   new(sync.RWMutex),
			length:    length,
		}, nil
	}
	datType, ok := fieldMap[field]
	if !ok {
		return nil, errors.Errorf("unrecognized field in trie")
	}
	fieldRoots, err := fieldConverters(field, []uint64{}, elements, true)
	if err != nil {
		return nil, err
	}
	if err := validateElements(field, elements, length); err != nil {
		return nil, err
	}
	switch datType {
	case basicArray:
		return &FieldTrie{
			fieldLayers: stateutil.ReturnTrieLayer(fieldRoots, length),
			field:       field,
			reference:   stateutil.NewRef(1),
			RWMutex:     new(sync.RWMutex),
			length:      length,
		}, nil
	case compositeArray:
		return &FieldTrie{
			fieldLayers: stateutil.ReturnTrieLayerVariable(fieldRoots, length),
			field:       field,
			reference:   stateutil.NewRef(1),
			RWMutex:     new(sync.RWMutex),
			length:      length,
		}, nil
	default:
		return nil, errors.Errorf("unrecognized data type in field map: %v", reflect.TypeOf(datType).Name())
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
	datType, ok := fieldMap[f.field]
	if !ok {
		return [32]byte{}, errors.Errorf("unrecognized field in trie")
	}
	fieldRoots, err := fieldConverters(f.field, indices, elements, false)
	if err != nil {
		return [32]byte{}, err
	}
	if err := f.validateIndices(indices); err != nil {
		return [32]byte{}, err
	}
	switch datType {
	case basicArray:
		fieldRoot, f.fieldLayers, err = stateutil.RecomputeFromLayer(fieldRoots, indices, f.fieldLayers)
		if err != nil {
			return [32]byte{}, err
		}
		return fieldRoot, nil
	case compositeArray:
		fieldRoot, f.fieldLayers, err = stateutil.RecomputeFromLayerVariable(fieldRoots, indices, f.fieldLayers)
		if err != nil {
			return [32]byte{}, err
		}
		return stateutil.AddInMixin(fieldRoot, uint64(len(f.fieldLayers[0])))
	default:
		return [32]byte{}, errors.Errorf("unrecognized data type in field map: %v", reflect.TypeOf(datType).Name())
	}

}

// CopyTrie copies the references to the elements the trie
// is built on.
func (f *FieldTrie) CopyTrie() *FieldTrie {
	if f.fieldLayers == nil {
		return &FieldTrie{
			field:     f.field,
			reference: stateutil.NewRef(1),
			RWMutex:   new(sync.RWMutex),
			length:    f.length,
		}
	}
	dstFieldTrie := make([][]*[32]byte, len(f.fieldLayers))
	for i, layer := range f.fieldLayers {
		dstFieldTrie[i] = make([]*[32]byte, len(layer))
		copy(dstFieldTrie[i], layer)
	}
	return &FieldTrie{
		fieldLayers: dstFieldTrie,
		field:       f.field,
		reference:   stateutil.NewRef(1),
		RWMutex:     new(sync.RWMutex),
		length:      f.length,
	}
}

// TrieRoot returns the corresponding root of the trie.
func (f *FieldTrie) TrieRoot() ([32]byte, error) {
	datType, ok := fieldMap[f.field]
	if !ok {
		return [32]byte{}, errors.Errorf("unrecognized field in trie")
	}
	switch datType {
	case basicArray:
		return *f.fieldLayers[len(f.fieldLayers)-1][0], nil
	case compositeArray:
		trieRoot := *f.fieldLayers[len(f.fieldLayers)-1][0]
		return stateutil.AddInMixin(trieRoot, uint64(len(f.fieldLayers[0])))
	default:
		return [32]byte{}, errors.Errorf("unrecognized data type in field map: %v", reflect.TypeOf(datType).Name())
	}
}

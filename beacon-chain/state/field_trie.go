package state

import (
	"reflect"
	"sync"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	"github.com/pkg/errors"

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

func NewFieldTrie(field fieldIndex, elements interface{}, length uint64) (*FieldTrie, error) {
	if elements == nil {
		return &FieldTrie{
			field:     field,
			reference: &reference{1},
			Mutex:     new(sync.Mutex),
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
	switch datType {
	case basicArray:
		return &FieldTrie{
			fieldLayers: stateutil.ReturnTrieLayer(fieldRoots, length),
			field:       field,
			reference:   &reference{1},
			Mutex:       new(sync.Mutex),
		}, nil
	case compositeArray:
		return &FieldTrie{
			fieldLayers: stateutil.ReturnTrieLayerVariable(fieldRoots, length),
			field:       field,
			reference:   &reference{1},
			Mutex:       new(sync.Mutex),
		}, nil
	default:
		return nil, errors.Errorf("unrecognized data type in field map: %v", reflect.TypeOf(datType).Name())
	}

}

func (f *FieldTrie) RecomputeTrie(indices []uint64, elements interface{}) ([32]byte, error) {
	f.Lock()
	defer f.Unlock()
	var fieldRoot [32]byte
	datType, ok := fieldMap[f.field]
	if !ok {
		return [32]byte{}, errors.Errorf("unrecognized field in trie")
	}
	fieldRoots, err := fieldConverters(f.field, indices, elements, false)
	if err != nil {
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

func fieldConverters(field fieldIndex, indices []uint64, elements interface{}, convertAll bool) ([][32]byte, error) {
	switch field {
	case blockRoots, stateRoots, randaoMixes:
		val, ok := elements.([][]byte)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([][]byte{}).Name(), reflect.TypeOf(elements).Name())
		}
		return handleByteArrays(val, indices, convertAll)
	case eth1DataVotes:
		val, ok := elements.([]*ethpb.Eth1Data)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.Eth1Data{}).Name(), reflect.TypeOf(elements).Name())
		}
		return handleEth1DataSlice(val, indices, convertAll)
	case validators:
		val, ok := elements.([]*ethpb.Validator)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.Validator{}).Name(), reflect.TypeOf(elements).Name())
		}
		return handleValidatorSlice(val, indices, convertAll)
	case previousEpochAttestations, currentEpochAttestations:
		val, ok := elements.([]*pb.PendingAttestation)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*pb.PendingAttestation{}).Name(), reflect.TypeOf(elements).Name())
		}
		return handlePendingAttestation(val, indices, convertAll)
	default:
		return [][32]byte{}, errors.Errorf("got unsupported type of %v", reflect.TypeOf(elements).Name())
	}
}

func handleByteArrays(val [][]byte, indices []uint64, convertAll bool) ([][32]byte, error) {
	roots := [][32]byte{}
	rootCreater := func(input []byte) {
		newRoot := bytesutil.ToBytes32(input)
		roots = append(roots, newRoot)
	}
	if convertAll {
		for i := range val {
			rootCreater(val[i])
		}
		return roots, nil
	}
	for _, idx := range indices {
		rootCreater(val[idx])
	}
	return roots, nil
}

func handleEth1DataSlice(val []*ethpb.Eth1Data, indices []uint64, convertAll bool) ([][32]byte, error) {
	roots := [][32]byte{}
	rootCreater := func(input *ethpb.Eth1Data) error {
		newRoot, err := stateutil.Eth1Root(input)
		if err != nil {
			return err
		}
		roots = append(roots, newRoot)
		return nil
	}
	if convertAll {
		for i := range val {
			err := rootCreater(val[i])
			if err != nil {
				return nil, err
			}
		}
		return roots, nil
	}
	for _, idx := range indices {
		err := rootCreater(val[idx])
		if err != nil {
			return nil, err
		}
	}
	return roots, nil
}

func handleValidatorSlice(val []*ethpb.Validator, indices []uint64, convertAll bool) ([][32]byte, error) {
	roots := [][32]byte{}
	rootCreater := func(input *ethpb.Validator) error {
		newRoot, err := stateutil.ValidatorRoot(input)
		if err != nil {
			return err
		}
		roots = append(roots, newRoot)
		return nil
	}
	if convertAll {
		for i := range val {
			err := rootCreater(val[i])
			if err != nil {
				return nil, err
			}
		}
		return roots, nil
	}
	for _, idx := range indices {
		err := rootCreater(val[idx])
		if err != nil {
			return nil, err
		}
	}
	return roots, nil
}

func handlePendingAttestation(val []*pb.PendingAttestation, indices []uint64, convertAll bool) ([][32]byte, error) {
	roots := [][32]byte{}
	rootCreater := func(input *pb.PendingAttestation) error {
		newRoot, err := stateutil.PendingAttestationRoot(input)
		if err != nil {
			return err
		}
		roots = append(roots, newRoot)
		return nil
	}
	if convertAll {
		for i := range val {
			err := rootCreater(val[i])
			if err != nil {
				return nil, err
			}
		}
		return roots, nil
	}
	for _, idx := range indices {
		err := rootCreater(val[idx])
		if err != nil {
			return nil, err
		}
	}
	return roots, nil
}

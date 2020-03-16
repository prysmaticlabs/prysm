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

func NewFieldTrie(field fieldIndex, elements [][]byte, length uint64) *FieldTrie {
	return &FieldTrie{
		fieldLayers: stateutil.ReturnTrieLayer(elements, length),
		field:       field,
		reference:   &reference{1},
		Mutex:       new(sync.Mutex),
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
	fieldRoots, err := fieldConverters(f.field, indices, elements)
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

func fieldConverters(field fieldIndex, indices []uint64, elements interface{}) ([][32]byte, error) {
	switch field {
	case blockRoots, stateRoots, randaoMixes:
		val, ok := elements.([][]byte)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([][]byte{}).Name(), reflect.TypeOf(elements).Name())
		}
		roots := [][32]byte{}
		for _, idx := range indices {
			roots = append(roots, bytesutil.ToBytes32(val[idx]))
		}
		return roots, nil
	case eth1DataVotes:
		val, ok := elements.([]*ethpb.Eth1Data)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.Eth1Data{}).Name(), reflect.TypeOf(elements).Name())
		}
		roots := [][32]byte{}
		for _, idx := range indices {
			newRoot, err := stateutil.Eth1Root(val[idx])
			if err != nil {
				return nil, err
			}
			roots = append(roots, newRoot)
		}
		return roots, nil
	case validators:
		val, ok := elements.([]*ethpb.Validator)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.Validator{}).Name(), reflect.TypeOf(elements).Name())
		}
		roots := [][32]byte{}
		for _, idx := range indices {
			newRoot, err := stateutil.ValidatorRoot(val[idx])
			if err != nil {
				return nil, err
			}
			roots = append(roots, newRoot)
		}
		return roots, nil
	case previousEpochAttestations, currentEpochAttestations:
		val, ok := elements.([]*pb.PendingAttestation)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*pb.PendingAttestation{}).Name(), reflect.TypeOf(elements).Name())
		}
		roots := [][32]byte{}
		for _, idx := range indices {
			newRoot, err := stateutil.PendingAttestationRoot(val[idx])
			if err != nil {
				return nil, err
			}
			roots = append(roots, newRoot)
		}
		return roots, nil
	default:
		return [][32]byte{}, errors.Errorf("got unsupported type of %v", reflect.TypeOf(elements).Name())
	}
}

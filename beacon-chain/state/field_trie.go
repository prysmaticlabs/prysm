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

func (f *FieldTrie) RecomputeTrieVariable(indices []uint64, elements [][32]byte) ([32]byte, error) {
	f.Lock()
	defer f.Unlock()
	var err error
	var fieldRoot [32]byte

	fieldRoot, f.fieldLayers, err = stateutil.RecomputeFromLayerVariable(elements, f.fieldLayers, indices)
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

func fieldConverters(field fieldIndex, elements interface{}) ([][32]byte, error) {
	switch field {
	case blockRoots, stateRoots, randaoMixes:
		val, ok := elements.([][32]byte)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([][32]byte{}).Name(), reflect.TypeOf(elements).Name())
		}
		return val, nil
	case eth1DataVotes:
		val, ok := elements.([]*ethpb.Eth1Data)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.Eth1Data{}).Name(), reflect.TypeOf(elements).Name())
		}
		roots := [][32]byte{}
		for i := range val {
			newRoot, err := stateutil.Eth1Root(val[i])
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
		for i := range val {
			newRoot, err := stateutil.ValidatorRoot(val[i])
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
		for i := range val {
			newRoot, err := stateutil.PendingAttestationRoot(val[i])
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

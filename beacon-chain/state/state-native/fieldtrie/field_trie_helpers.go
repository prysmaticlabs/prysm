package fieldtrie

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/fieldtrie"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

func (f *FieldTrie) validateIndices(idxs []uint64) error {
	length := f.length
	if f.dataType == types.CompressedArray {
		comLength, err := f.field.ElemsInChunk()
		if err != nil {
			return err
		}
		length *= comLength
	}
	for _, idx := range idxs {
		if idx >= length {
			return errors.Errorf("invalid index for field %s: %d >= length %d", f.field.String(version.Phase0), idx, length)
		}
	}
	return nil
}

func validateElements(field types.FieldIndex, dataType types.DataType, elements interface{}, length uint64) error {
	if dataType == types.CompressedArray {
		comLength, err := field.ElemsInChunk()
		if err != nil {
			return err
		}
		length *= comLength
	}
	val := reflect.Indirect(reflect.ValueOf(elements))
	if val.Len() > int(length) {
		return errors.Errorf("elements length is larger than expected for field %s: %d > %d", field.String(version.Phase0), val.Len(), length)
	}
	return nil
}

// fieldConverters converts the corresponding field and the provided elements to the appropriate roots.
func fieldConverters(field types.FieldIndex, indices []uint64, elements interface{}, convertAll bool) ([][32]byte, error) {
	switch field {
	case types.BlockRoots:
		val, ok := elements.(*customtypes.BlockRoots)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([][]byte{}).Name(), reflect.TypeOf(elements).Name())
		}
		roots := make([][]byte, len(val))
		for i, r := range val {
			tmp := r
			roots[i] = tmp[:]
		}
		return fieldtrie.HandleByteArrays(roots, indices, convertAll)
	case types.StateRoots:
		val, ok := elements.(*customtypes.StateRoots)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([][]byte{}).Name(), reflect.TypeOf(elements).Name())
		}
		roots := make([][]byte, len(val))
		for i, r := range val {
			tmp := r
			roots[i] = tmp[:]
		}
		return fieldtrie.HandleByteArrays(roots, indices, convertAll)
	case types.RandaoMixes:
		val, ok := elements.(*customtypes.RandaoMixes)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([][]byte{}).Name(), reflect.TypeOf(elements).Name())
		}
		mixes := make([][]byte, len(val))
		for i, m := range val {
			tmp := m
			mixes[i] = tmp[:]
		}
		return fieldtrie.HandleByteArrays(mixes, indices, convertAll)
	case types.Eth1DataVotes:
		val, ok := elements.([]*ethpb.Eth1Data)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.Eth1Data{}).Name(), reflect.TypeOf(elements).Name())
		}
		return fieldtrie.HandleEth1DataSlice(val, indices, convertAll)
	case types.Validators:
		val, ok := elements.([]*ethpb.Validator)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.Validator{}).Name(), reflect.TypeOf(elements).Name())
		}
		return fieldtrie.HandleValidatorSlice(val, indices, convertAll)
	case types.PreviousEpochAttestations, types.CurrentEpochAttestations:
		val, ok := elements.([]*ethpb.PendingAttestation)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.PendingAttestation{}).Name(), reflect.TypeOf(elements).Name())
		}
		return fieldtrie.HandlePendingAttestationSlice(val, indices, convertAll)
	case types.Balances:
		val, ok := elements.([]uint64)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]uint64{}).Name(), reflect.TypeOf(elements).Name())
		}
		return fieldtrie.HandleBalanceSlice(val, indices, convertAll)
	default:
		return [][32]byte{}, errors.Errorf("got unsupported type of %v", reflect.TypeOf(elements).Name())
	}
}

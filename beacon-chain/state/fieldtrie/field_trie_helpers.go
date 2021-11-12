package fieldtrie

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

func (f *FieldTrie) validateIndices(idxs []uint64) error {
	for _, idx := range idxs {
		if idx >= f.length {
			return errors.Errorf("invalid index for field %s: %d >= length %d", f.field.String(version.Phase0), idx, f.length)
		}
	}
	return nil
}

func validateElements(field types.FieldIndex, elements interface{}, length uint64) error {
	val := reflect.ValueOf(elements)
	if val.Len() > int(length) {
		return errors.Errorf("elements length is larger than expected for field %s: %d > %d", field.String(version.Phase0), val.Len(), length)
	}
	return nil
}

// fieldConverters converts the corresponding field and the provided elements to the appropriate roots.
func fieldConverters(field types.FieldIndex, indices []uint64, elements interface{}, convertAll bool) ([][32]byte, error) {
	switch field {
	case types.BlockRoots, types.StateRoots:
		val, ok := elements.(customtypes.StateRoots)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([][]byte{}).Name(), reflect.TypeOf(elements).Name())
		}
		roots := make([][]byte, len(val))
		for i, r := range val {
			tmp := r
			roots[i] = tmp[:]
		}
		return stateutil.HandleByteArrays(roots, indices, convertAll)
	case types.RandaoMixes:
		val, ok := elements.(customtypes.RandaoMixes)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([][]byte{}).Name(), reflect.TypeOf(elements).Name())
		}
		mixes := make([][]byte, len(val))
		for i, m := range val {
			tmp := m
			mixes[i] = tmp[:]
		}
		return stateutil.HandleByteArrays(mixes, indices, convertAll)
	case types.Eth1DataVotes:
		val, ok := elements.([]*ethpb.Eth1Data)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.Eth1Data{}).Name(), reflect.TypeOf(elements).Name())
		}
		return HandleEth1DataSlice(val, indices, convertAll)
	case types.Validators:
		val, ok := elements.([]*ethpb.Validator)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.Validator{}).Name(), reflect.TypeOf(elements).Name())
		}
		return stateutil.HandleValidatorSlice(val, indices, convertAll)
	case types.PreviousEpochAttestations, types.CurrentEpochAttestations:
		val, ok := elements.([]*ethpb.PendingAttestation)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]*ethpb.PendingAttestation{}).Name(), reflect.TypeOf(elements).Name())
		}
		return handlePendingAttestation(val, indices, convertAll)
	default:
		return [][32]byte{}, errors.Errorf("got unsupported type of %v", reflect.TypeOf(elements).Name())
	}
}

// HandleEth1DataSlice processes a list of eth1data and indices into the appropriate roots.
func HandleEth1DataSlice(val []*ethpb.Eth1Data, indices []uint64, convertAll bool) ([][32]byte, error) {
	length := len(indices)
	if convertAll {
		length = len(val)
	}
	roots := make([][32]byte, 0, length)
	hasher := hash.CustomSHA256Hasher()
	rootCreator := func(input *ethpb.Eth1Data) error {
		newRoot, err := stateutil.Eth1DataRootWithHasher(hasher, input)
		if err != nil {
			return err
		}
		roots = append(roots, newRoot)
		return nil
	}
	if convertAll {
		for i := range val {
			err := rootCreator(val[i])
			if err != nil {
				return nil, err
			}
		}
		return roots, nil
	}
	if len(val) > 0 {
		for _, idx := range indices {
			if idx > uint64(len(val))-1 {
				return nil, fmt.Errorf("index %d greater than number of items in eth1 data slice %d", idx, len(val))
			}
			err := rootCreator(val[idx])
			if err != nil {
				return nil, err
			}
		}
	}
	return roots, nil
}

func handlePendingAttestation(val []*ethpb.PendingAttestation, indices []uint64, convertAll bool) ([][32]byte, error) {
	length := len(indices)
	if convertAll {
		length = len(val)
	}
	roots := make([][32]byte, 0, length)
	hasher := hash.CustomSHA256Hasher()
	rootCreator := func(input *ethpb.PendingAttestation) error {
		newRoot, err := stateutil.PendingAttRootWithHasher(hasher, input)
		if err != nil {
			return err
		}
		roots = append(roots, newRoot)
		return nil
	}
	if convertAll {
		for i := range val {
			err := rootCreator(val[i])
			if err != nil {
				return nil, err
			}
		}
		return roots, nil
	}
	if len(val) > 0 {
		for _, idx := range indices {
			if idx > uint64(len(val))-1 {
				return nil, fmt.Errorf("index %d greater than number of pending attestations %d", idx, len(val))
			}
			err := rootCreator(val[idx])
			if err != nil {
				return nil, err
			}
		}
	}
	return roots, nil
}

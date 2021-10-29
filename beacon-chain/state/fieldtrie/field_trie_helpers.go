package fieldtrie

import (
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
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
	case types.BlockRoots, types.StateRoots, types.RandaoMixes:
		val, ok := elements.([][]byte)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([][]byte{}).Name(), reflect.TypeOf(elements).Name())
		}
		return stateutil.HandleByteArrays(val, indices, convertAll)
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
	case types.Balances:
		val, ok := elements.([]uint64)
		if !ok {
			return nil, errors.Errorf("Wanted type of %v but got %v",
				reflect.TypeOf([]uint64{}).Name(), reflect.TypeOf(elements).Name())
		}
		return handleBalanceSlice(val, indices, convertAll)
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

func handleBalanceSlice(val []uint64, indices []uint64, convertAll bool) ([][32]byte, error) {
	if convertAll {
		balancesMarshaling := make([][]byte, 0)
		for i := 0; i < len(val); i++ {
			balanceBuf := make([]byte, 8)
			binary.LittleEndian.PutUint64(balanceBuf, val[i])
			balancesMarshaling = append(balancesMarshaling, balanceBuf)
		}
		balancesChunks, err := ssz.PackByChunk(balancesMarshaling)
		if err != nil {
			return [][32]byte{}, errors.Wrap(err, "could not pack balances into chunks")
		}
		return balancesChunks, nil
	}
	if len(val) > 0 {
		numOfElems, err := types.Balances.CompressedLength()
		if err != nil {
			return nil, err
		}
		roots := [][32]byte{}
		for _, idx := range indices {
			// We split the indexes into their relevant groups. Balances
			// are compressed according to 4 values -> 1 chunk.
			startIdx := idx / numOfElems
			startGroup := startIdx * numOfElems
			chunk := [32]byte{}
			sizeOfElem := len(chunk) / int(numOfElems)
			for i, j := 0, startGroup; j < startGroup+numOfElems; i, j = i+sizeOfElem, j+1 {
				wantedVal := uint64(0)
				// We are adding chunks in sets of 4, if the set is at the edge of the array
				// then you will need to zero out the rest of the chunk. Ex : 41 indexes,
				// so 41 % 4 = 1 . There are 3 indexes, which do not exist yet but we
				// have to add in as a root. These 3 indexes are then given a 'zero' value.
				if int(j) < len(val) {
					wantedVal = val[j]
				}
				binary.LittleEndian.PutUint64(chunk[i:i+sizeOfElem], wantedVal)
			}
			roots = append(roots, chunk)
		}
		return roots, nil
	}
	return [][32]byte{}, nil
}

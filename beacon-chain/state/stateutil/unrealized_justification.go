package stateutil

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	multi_value_slice "github.com/prysmaticlabs/prysm/v5/container/multi-value-slice"
	"github.com/prysmaticlabs/prysm/v5/math"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// UnrealizedCheckpointBalances returns the total current active balance, the
// total previous epoch correctly attested for target balance, and the total
// current epoch correctly attested for target balance. It takes the current and
// previous epoch participation bits as parameters so implicitly only works for
// beacon states post-Altair.
func UnrealizedCheckpointBalances(cp, pp []byte, validators ValReader, currentEpoch primitives.Epoch) (uint64, uint64, uint64, error) {
	targetIdx := params.BeaconConfig().TimelyTargetFlagIndex
	activeBalance := uint64(0)
	currentTarget := uint64(0)
	prevTarget := uint64(0)
	if len(cp) < validators.Len() || len(pp) < validators.Len() {
		return 0, 0, 0, errors.New("participation does not match validator set")
	}

	valLength := validators.Len()
	for i := 0; i < valLength; i++ {
		v, err := validators.At(i)
		if err != nil {
			return 0, 0, 0, err
		}
		active := v.ActivationEpoch <= currentEpoch && currentEpoch < v.ExitEpoch
		if active && !v.Slashed {
			activeBalance, err = math.Add64(activeBalance, v.EffectiveBalance)
			if err != nil {
				return 0, 0, 0, err
			}
			if ((cp[i] >> targetIdx) & 1) == 1 {
				currentTarget, err = math.Add64(currentTarget, v.EffectiveBalance)
				if err != nil {
					return 0, 0, 0, err
				}
			}
			if ((pp[i] >> targetIdx) & 1) == 1 {
				prevTarget, err = math.Add64(prevTarget, v.EffectiveBalance)
				if err != nil {
					return 0, 0, 0, err
				}
			}
		}
	}
	return activeBalance, prevTarget, currentTarget, nil
}

// ValReader specifies an interface through which we can access the validator registry.
type ValReader interface {
	Len() int
	At(i int) (*ethpb.Validator, error)
}

// ValSliceReader describes a struct that conforms to the ValReader interface
type ValSliceReader struct {
	Validators []*ethpb.Validator
}

// NewValSliceReader constructs a ValSliceReader object.
func NewValSliceReader(vals []*ethpb.Validator) ValSliceReader {
	return ValSliceReader{Validators: vals}
}

// Len is the length of the validator registry.
func (v ValSliceReader) Len() int {
	return len(v.Validators)
}

// At returns the validator at the provided index.
func (v ValSliceReader) At(i int) (*ethpb.Validator, error) {
	return v.Validators[i], nil
}

// ValMultiValueSliceReader describes a struct that conforms to the ValReader interface.
// This struct is specifically designed for accessing validator data from a
// multivalue slice.
type ValMultiValueSliceReader struct {
	ValMVSlice *multi_value_slice.Slice[*ethpb.Validator]
	Identifier multi_value_slice.Identifiable
}

// NewValMultiValueSliceReader constructs a new val reader object.
func NewValMultiValueSliceReader(valSlice *multi_value_slice.Slice[*ethpb.Validator],
	identifier multi_value_slice.Identifiable) ValMultiValueSliceReader {
	return ValMultiValueSliceReader{
		ValMVSlice: valSlice,
		Identifier: identifier,
	}
}

// Len is the length of the validator registry.
func (v ValMultiValueSliceReader) Len() int {
	return v.ValMVSlice.Len(v.Identifier)
}

// At returns the validator at the provided index.
func (v ValMultiValueSliceReader) At(i int) (*ethpb.Validator, error) {
	return v.ValMVSlice.At(v.Identifier, uint64(i))
}

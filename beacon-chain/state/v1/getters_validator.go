package v1

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// ValidatorIndexOutOfRangeError represents an error scenario where a validator does not exist
// at a given index in the validator's array.
type ValidatorIndexOutOfRangeError struct {
	message string
}

// NewValidatorIndexOutOfRangeError creates a new error instance.
func NewValidatorIndexOutOfRangeError(index types.ValidatorIndex) ValidatorIndexOutOfRangeError {
	return ValidatorIndexOutOfRangeError{
		message: fmt.Sprintf("index %d out of range", index),
	}
}

// Error returns the underlying error message.
func (e *ValidatorIndexOutOfRangeError) Error() string {
	return e.message
}

// Validators participating in consensus on the beacon chain.
func (b *BeaconState) Validators() []*ethpb.Validator {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.validators()
}

// validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) validators() []*ethpb.Validator {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	res := make([]*ethpb.Validator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		val := b.state.Validators[i]
		if val == nil {
			continue
		}
		res[i] = ethpb.CopyValidator(val)
	}
	return res
}

// references of validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState. This does not
// copy fully and instead just copies the reference.
func (b *BeaconState) validatorsReferences() []*ethpb.Validator {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	res := make([]*ethpb.Validator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		validator := b.state.Validators[i]
		if validator == nil {
			continue
		}
		// copy validator reference instead.
		res[i] = validator
	}
	return res
}

// ValidatorAtIndex is the validator at the provided index.
func (b *BeaconState) ValidatorAtIndex(idx types.ValidatorIndex) (*ethpb.Validator, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.Validators == nil {
		return &ethpb.Validator{}, nil
	}
	if uint64(len(b.state.Validators)) <= uint64(idx) {
		e := NewValidatorIndexOutOfRangeError(idx)
		return nil, &e
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	val := b.state.Validators[idx]
	return ethpb.CopyValidator(val), nil
}

// ValidatorAtIndexReadOnly is the validator at the provided index. This method
// doesn't clone the validator.
func (b *BeaconState) ValidatorAtIndexReadOnly(idx types.ValidatorIndex) (state.ReadOnlyValidator, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.Validators == nil {
		return nil, state.ErrNilValidatorsInState
	}
	if uint64(len(b.state.Validators)) <= uint64(idx) {
		e := NewValidatorIndexOutOfRangeError(idx)
		return nil, &e
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return NewValidator(b.state.Validators[idx])
}

// ValidatorIndexByPubkey returns a given validator by its 48-byte public key.
func (b *BeaconState) ValidatorIndexByPubkey(key [fieldparams.BLSPubkeyLength]byte) (types.ValidatorIndex, bool) {
	if b == nil || b.valMapHandler == nil || b.valMapHandler.IsNil() {
		return 0, false
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	numOfVals := len(b.state.Validators)

	idx, ok := b.valMapHandler.Get(key)
	if ok && types.ValidatorIndex(numOfVals) <= idx {
		return types.ValidatorIndex(0), false
	}
	return idx, ok
}

// PubkeyAtIndex returns the pubkey at the given
// validator index.
func (b *BeaconState) PubkeyAtIndex(idx types.ValidatorIndex) [fieldparams.BLSPubkeyLength]byte {
	if !b.hasInnerState() {
		return [fieldparams.BLSPubkeyLength]byte{}
	}
	if uint64(idx) >= uint64(len(b.state.Validators)) {
		return [fieldparams.BLSPubkeyLength]byte{}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.state.Validators[idx] == nil {
		return [fieldparams.BLSPubkeyLength]byte{}
	}
	return bytesutil.ToBytes48(b.state.Validators[idx].PublicKey)
}

// NumValidators returns the size of the validator registry.
func (b *BeaconState) NumValidators() int {
	if !b.hasInnerState() {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.state.Validators)
}

// ReadFromEveryValidator reads values from every validator and applies it to the provided function.
//
// WARNING: This method is potentially unsafe, as it exposes the actual validator registry.
func (b *BeaconState) ReadFromEveryValidator(f func(idx int, val state.ReadOnlyValidator) error) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	if b.state.Validators == nil {
		return errors.New("nil validators in state")
	}
	b.lock.RLock()
	validators := b.state.Validators
	b.lock.RUnlock()

	for i, v := range validators {
		v, err := NewValidator(v)
		if err != nil {
			return err
		}
		if err := f(i, v); err != nil {
			return err
		}
	}
	return nil
}

// Balances of validators participating in consensus on the beacon chain.
func (b *BeaconState) Balances() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Balances == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.balances()
}

// balances of validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) balances() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Balances == nil {
		return nil
	}

	res := make([]uint64, len(b.state.Balances))
	copy(res, b.state.Balances)
	return res
}

// BalanceAtIndex of validator with the provided index.
func (b *BeaconState) BalanceAtIndex(idx types.ValidatorIndex) (uint64, error) {
	if !b.hasInnerState() {
		return 0, ErrNilInnerState
	}
	if b.state.Balances == nil {
		return 0, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if uint64(len(b.state.Balances)) <= uint64(idx) {
		return 0, fmt.Errorf("index of %d does not exist", idx)
	}
	return b.state.Balances[idx], nil
}

// BalancesLength returns the length of the balances slice.
func (b *BeaconState) BalancesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.Balances == nil {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.balancesLength()
}

// Slashings of validators on the beacon chain.
func (b *BeaconState) Slashings() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Slashings == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slashings()
}

// slashings of validators on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) slashings() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Slashings == nil {
		return nil
	}

	res := make([]uint64, len(b.state.Slashings))
	copy(res, b.state.Slashings)
	return res
}

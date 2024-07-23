package state_native

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// Validators participating in consensus on the beacon chain.
func (b *BeaconState) Validators() []*ethpb.Validator {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.validatorsVal()
}

// ValidatorsReadOnly participating in consensus on the beacon chain.
func (b *BeaconState) ValidatorsReadOnly() []state.ReadOnlyValidator {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.validatorsReadOnlyVal()
}

func (b *BeaconState) validatorsVal() []*ethpb.Validator {
	var v []*ethpb.Validator
	if features.Get().EnableExperimentalState {
		if b.validatorsMultiValue == nil {
			return nil
		}
		v = b.validatorsMultiValue.Value(b)
	} else {
		if b.validators == nil {
			return nil
		}
		v = b.validators
	}

	res := make([]*ethpb.Validator, len(v))
	for i := 0; i < len(res); i++ {
		val := v[i]
		if val == nil {
			continue
		}
		res[i] = ethpb.CopyValidator(val)
	}
	return res
}

func (b *BeaconState) validatorsReadOnlyVal() []state.ReadOnlyValidator {
	var v []*ethpb.Validator
	if features.Get().EnableExperimentalState {
		if b.validatorsMultiValue == nil {
			return nil
		}
		v = b.validatorsMultiValue.Value(b)
	} else {
		if b.validators == nil {
			return nil
		}
		v = b.validators
	}

	res := make([]state.ReadOnlyValidator, len(v))
	var err error
	for i := 0; i < len(res); i++ {
		val := v[i]
		if val == nil {
			continue
		}
		res[i], err = NewValidator(val)
		if err != nil {
			continue
		}
	}
	return res
}

// references of validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState. This does not
// copy fully and instead just copies the reference.
func (b *BeaconState) validatorsReferences() []*ethpb.Validator {
	if b.validators == nil {
		return nil
	}

	res := make([]*ethpb.Validator, len(b.validators), len(b.validators)+int(params.BeaconConfig().MaxDeposits))
	for i := 0; i < len(res); i++ {
		validator := b.validators[i]
		if validator == nil {
			continue
		}
		// copy validator reference instead.
		res[i] = validator
	}
	return res
}

func (b *BeaconState) validatorsLen() int {
	if features.Get().EnableExperimentalState {
		if b.validatorsMultiValue == nil {
			return 0
		}
		return b.validatorsMultiValue.Len(b)
	}
	return len(b.validators)
}

// ValidatorAtIndex is the validator at the provided index.
func (b *BeaconState) ValidatorAtIndex(idx primitives.ValidatorIndex) (*ethpb.Validator, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.validatorAtIndex(idx)
}

func (b *BeaconState) validatorAtIndex(idx primitives.ValidatorIndex) (*ethpb.Validator, error) {
	if features.Get().EnableExperimentalState {
		if b.validatorsMultiValue == nil {
			return &ethpb.Validator{}, nil
		}
		v, err := b.validatorsMultiValue.At(b, uint64(idx))
		if err != nil {
			return nil, err
		}
		return ethpb.CopyValidator(v), nil
	}

	if b.validators == nil {
		return &ethpb.Validator{}, nil
	}
	if uint64(len(b.validators)) <= uint64(idx) {
		return nil, errors.Wrapf(consensus_types.ErrOutOfBounds, "validator index %d does not exist", idx)
	}
	val := b.validators[idx]
	return ethpb.CopyValidator(val), nil
}

// ValidatorAtIndexReadOnly is the validator at the provided index. This method
// doesn't clone the validator.
func (b *BeaconState) ValidatorAtIndexReadOnly(idx primitives.ValidatorIndex) (state.ReadOnlyValidator, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if features.Get().EnableExperimentalState {
		if b.validatorsMultiValue == nil {
			return nil, state.ErrNilValidatorsInState
		}
		v, err := b.validatorsMultiValue.At(b, uint64(idx))
		if err != nil {
			return nil, err
		}
		return NewValidator(v)
	}

	if b.validators == nil {
		return nil, state.ErrNilValidatorsInState
	}
	if uint64(len(b.validators)) <= uint64(idx) {
		return nil, errors.Wrapf(consensus_types.ErrOutOfBounds, "validator index %d does not exist", idx)
	}
	val := b.validators[idx]
	return NewValidator(val)
}

// ValidatorIndexByPubkey returns a given validator by its 48-byte public key.
func (b *BeaconState) ValidatorIndexByPubkey(key [fieldparams.BLSPubkeyLength]byte) (primitives.ValidatorIndex, bool) {
	if b == nil || b.valMapHandler == nil || b.valMapHandler.IsNil() {
		return 0, false
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.Version() >= version.Electra {
		return b.getValidatorIndex(key)
	}

	var numOfVals int
	if features.Get().EnableExperimentalState {
		numOfVals = b.validatorsMultiValue.Len(b)
	} else {
		numOfVals = len(b.validators)
	}

	idx, ok := b.valMapHandler.Get(key)
	if ok && primitives.ValidatorIndex(numOfVals) <= idx {
		return primitives.ValidatorIndex(0), false
	}
	return idx, ok
}

// PubkeyAtIndex returns the pubkey at the given
// validator index.
func (b *BeaconState) PubkeyAtIndex(idx primitives.ValidatorIndex) [fieldparams.BLSPubkeyLength]byte {
	b.lock.RLock()
	defer b.lock.RUnlock()

	var v *ethpb.Validator
	if features.Get().EnableExperimentalState {
		var err error
		v, err = b.validatorsMultiValue.At(b, uint64(idx))
		if err != nil {
			return [fieldparams.BLSPubkeyLength]byte{}
		}
	} else {
		if uint64(idx) >= uint64(len(b.validators)) {
			return [fieldparams.BLSPubkeyLength]byte{}
		}
		v = b.validators[idx]
	}

	if v == nil {
		return [fieldparams.BLSPubkeyLength]byte{}
	}
	return bytesutil.ToBytes48(v.PublicKey)
}

// AggregateKeyFromIndices builds an aggregated public key from the provided
// validator indices.
func (b *BeaconState) AggregateKeyFromIndices(idxs []uint64) (bls.PublicKey, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	pubKeys := make([][]byte, len(idxs))
	for i, idx := range idxs {
		var v *ethpb.Validator
		if features.Get().EnableExperimentalState {
			var err error
			v, err = b.validatorsMultiValue.At(b, idx)
			if err != nil {
				return nil, err
			}
		} else {
			if idx >= uint64(len(b.validators)) {
				return nil, consensus_types.ErrOutOfBounds
			}
			v = b.validators[idx]
		}

		if v == nil {
			return nil, ErrNilWrappedValidator
		}
		pubKeys[i] = v.PublicKey
	}
	return bls.AggregatePublicKeys(pubKeys)
}

// PublicKeys builds a list of all validator public keys, with each key's index aligned to its validator index.
func (b *BeaconState) PublicKeys() ([][fieldparams.BLSPubkeyLength]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	l := b.validatorsLen()
	res := make([][fieldparams.BLSPubkeyLength]byte, l)
	for i := 0; i < l; i++ {
		if features.Get().EnableExperimentalState {
			val, err := b.validatorsMultiValue.At(b, uint64(i))
			if err != nil {
				return nil, err
			}
			copy(res[i][:], val.PublicKey)
		} else {
			copy(res[i][:], b.validators[i].PublicKey)
		}
	}
	return res, nil
}

// NumValidators returns the size of the validator registry.
func (b *BeaconState) NumValidators() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.validatorsLen()
}

// ReadFromEveryValidator reads values from every validator and applies it to the provided function.
//
// WARNING: This method is potentially unsafe, as it exposes the actual validator registry.
func (b *BeaconState) ReadFromEveryValidator(f func(idx int, val state.ReadOnlyValidator) error) error {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if features.Get().EnableExperimentalState {
		return b.readFromEveryValidatorMVSlice(f)
	}

	if b.validators == nil {
		return state.ErrNilValidatorsInState
	}

	validators := b.validators

	for i, v := range validators {
		v, err := NewValidator(v)
		if err != nil {
			return err
		}
		if err = f(i, v); err != nil {
			return err
		}
	}
	return nil
}

// WARNING: This function works only for the multi-value slice feature.
func (b *BeaconState) readFromEveryValidatorMVSlice(f func(idx int, val state.ReadOnlyValidator) error) error {
	if b.validatorsMultiValue == nil {
		return state.ErrNilValidatorsInState
	}
	l := b.validatorsMultiValue.Len(b)
	for i := 0; i < l; i++ {
		v, err := b.validatorsMultiValue.At(b, uint64(i))
		if err != nil {
			return err
		}
		rov, err := NewValidator(v)
		if err != nil {
			return err
		}
		if err = f(i, rov); err != nil {
			return err
		}
	}
	return nil
}

// Balances of validators participating in consensus on the beacon chain.
func (b *BeaconState) Balances() []uint64 {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.balancesVal()
}

func (b *BeaconState) balancesVal() []uint64 {
	if features.Get().EnableExperimentalState {
		if b.balancesMultiValue == nil {
			return nil
		}
		return b.balancesMultiValue.Value(b)
	}
	if b.balances == nil {
		return nil
	}
	res := make([]uint64, len(b.balances))
	copy(res, b.balances)
	return res
}

// BalanceAtIndex of validator with the provided index.
func (b *BeaconState) BalanceAtIndex(idx primitives.ValidatorIndex) (uint64, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.balanceAtIndex(idx)
}

func (b *BeaconState) balanceAtIndex(idx primitives.ValidatorIndex) (uint64, error) {
	if features.Get().EnableExperimentalState {
		if b.balancesMultiValue == nil {
			return 0, nil
		}
		return b.balancesMultiValue.At(b, uint64(idx))
	}
	if b.balances == nil {
		return 0, nil
	}
	if uint64(len(b.balances)) <= uint64(idx) {
		return 0, errors.Wrapf(consensus_types.ErrOutOfBounds, "balance index %d does not exist", idx)
	}
	return b.balances[idx], nil
}

// BalancesLength returns the length of the balances slice.
func (b *BeaconState) BalancesLength() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if features.Get().EnableExperimentalState {
		if b.balancesMultiValue == nil {
			return 0
		}
		return b.balancesMultiValue.Len(b)
	}
	return len(b.balances)
}

// Slashings of validators on the beacon chain.
func (b *BeaconState) Slashings() []uint64 {
	if b.slashings == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slashingsVal()
}

// slashingsVal of validators on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) slashingsVal() []uint64 {
	if b.slashings == nil {
		return nil
	}

	res := make([]uint64, len(b.slashings))
	copy(res, b.slashings)
	return res
}

// InactivityScores of validators participating in consensus on the beacon chain.
func (b *BeaconState) InactivityScores() ([]uint64, error) {
	if b.version == version.Phase0 {
		return nil, errNotSupported("InactivityScores", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.inactivityScoresVal(), nil
}

func (b *BeaconState) inactivityScoresVal() []uint64 {
	if features.Get().EnableExperimentalState {
		if b.inactivityScoresMultiValue == nil {
			return nil
		}
		return b.inactivityScoresMultiValue.Value(b)
	}
	if b.inactivityScores == nil {
		return nil
	}
	res := make([]uint64, len(b.inactivityScores))
	copy(res, b.inactivityScores)
	return res
}

// ActiveBalanceAtIndex returns the active balance for the given validator.
//
// Spec definition:
//
//	def get_active_balance(state: BeaconState, validator_index: ValidatorIndex) -> Gwei:
//	    max_effective_balance = get_validator_max_effective_balance(state.validators[validator_index])
//	    return min(state.balances[validator_index], max_effective_balance)
func (b *BeaconState) ActiveBalanceAtIndex(i primitives.ValidatorIndex) (uint64, error) {
	if b.version < version.Electra {
		return 0, errNotSupported("ActiveBalanceAtIndex", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	v, err := b.validatorAtIndex(i)
	if err != nil {
		return 0, err
	}

	bal, err := b.balanceAtIndex(i)
	if err != nil {
		return 0, err
	}

	return min(bal, helpers.ValidatorMaxEffectiveBalance(v)), nil
}

// PendingBalanceToWithdraw returns the sum of all pending withdrawals for the given validator.
//
// Spec definition:
//
//	def get_pending_balance_to_withdraw(state: BeaconState, validator_index: ValidatorIndex) -> Gwei:
//	    return sum(
//	        withdrawal.amount for withdrawal in state.pending_partial_withdrawals if withdrawal.index == validator_index)
func (b *BeaconState) PendingBalanceToWithdraw(idx primitives.ValidatorIndex) (uint64, error) {
	if b.version < version.Electra {
		return 0, errNotSupported("PendingBalanceToWithdraw", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	// TODO: Consider maintaining this value in the state, if it's a potential bottleneck.
	// This is n*m complexity, but this method can only be called
	// MAX_WITHDRAWAL_REQUESTS_PER_PAYLOAD per slot. A more optimized storage indexing such as a
	// lookup map could be used to reduce the complexity marginally.
	var sum uint64
	for _, w := range b.pendingPartialWithdrawals {
		if w.Index == idx {
			sum += w.Amount
		}
	}
	return sum, nil
}

func (b *BeaconState) HasPendingBalanceToWithdraw(idx primitives.ValidatorIndex) (bool, error) {
	if b.version < version.Electra {
		return false, errNotSupported("HasPendingBalanceToWithdraw", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	// TODO: Consider maintaining this value in the state, if it's a potential bottleneck.
	// This is n*m complexity, but this method can only be called
	// MAX_WITHDRAWAL_REQUESTS_PER_PAYLOAD per slot. A more optimized storage indexing such as a
	// lookup map could be used to reduce the complexity marginally.
	for _, w := range b.pendingPartialWithdrawals {
		if w.Index == idx {
			return true, nil
		}
	}

	return false, nil
}

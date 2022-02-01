package v3

import (
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SetValidators for the beacon state. Updates the entire
// to a new value by overwriting the previous one.
func (b *BeaconState) SetValidators(val []*ethpb.Validator) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.validators = val
	b.sharedFieldReferences[validators].MinusRef()
	b.sharedFieldReferences[validators] = stateutil.NewRef(1)
	b.markFieldAsDirty(validators)
	b.rebuildTrie[validators] = true
	b.valMapHandler = stateutil.NewValMapHandler(b.validators)
	return nil
}

// ApplyToEveryValidator applies the provided callback function to each validator in the
// validator registry.
func (b *BeaconState) ApplyToEveryValidator(f func(idx int, val *ethpb.Validator) (bool, *ethpb.Validator, error)) error {
	b.lock.Lock()
	v := b.validators
	if ref := b.sharedFieldReferences[validators]; ref.Refs() > 1 {
		v = b.validatorsReferences()
		ref.MinusRef()
		b.sharedFieldReferences[validators] = stateutil.NewRef(1)
	}
	b.lock.Unlock()
	var changedVals []uint64
	for i, val := range v {
		changed, newVal, err := f(i, val)
		if err != nil {
			return err
		}
		if changed {
			changedVals = append(changedVals, uint64(i))
			v[i] = newVal
		}
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.validators = v
	b.markFieldAsDirty(validators)
	b.addDirtyIndices(validators, changedVals)

	return nil
}

// UpdateValidatorAtIndex for the beacon state. Updates the validator
// at a specific index to a new value.
func (b *BeaconState) UpdateValidatorAtIndex(idx types.ValidatorIndex, val *ethpb.Validator) error {
	if uint64(len(b.validators)) <= uint64(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	v := b.validators
	if ref := b.sharedFieldReferences[validators]; ref.Refs() > 1 {
		v = b.validatorsReferences()
		ref.MinusRef()
		b.sharedFieldReferences[validators] = stateutil.NewRef(1)
	}

	v[idx] = val
	b.validators = v
	b.markFieldAsDirty(validators)
	b.addDirtyIndices(validators, []uint64{uint64(idx)})

	return nil
}

// SetBalances for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetBalances(val []uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[balances].MinusRef()
	b.sharedFieldReferences[balances] = stateutil.NewRef(1)

	b.balances = val
	b.markFieldAsDirty(balances)
	b.rebuildTrie[balances] = true
	return nil
}

// UpdateBalancesAtIndex for the beacon state. This method updates the balance
// at a specific index to a new value.
func (b *BeaconState) UpdateBalancesAtIndex(idx types.ValidatorIndex, val uint64) error {
	if uint64(len(b.balances)) <= uint64(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	bals := b.balances
	if b.sharedFieldReferences[balances].Refs() > 1 {
		bals = b.balancesVal()
		b.sharedFieldReferences[balances].MinusRef()
		b.sharedFieldReferences[balances] = stateutil.NewRef(1)
	}

	bals[idx] = val
	b.balances = bals
	b.markFieldAsDirty(balances)
	b.addDirtyIndices(balances, []uint64{uint64(idx)})
	return nil
}

// SetSlashings for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetSlashings(val []uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[slashings].MinusRef()
	b.sharedFieldReferences[slashings] = stateutil.NewRef(1)

	b.slashings = val
	b.markFieldAsDirty(slashings)
	return nil
}

// UpdateSlashingsAtIndex for the beacon state. Updates the slashings
// at a specific index to a new value.
func (b *BeaconState) UpdateSlashingsAtIndex(idx, val uint64) error {
	if uint64(len(b.slashings)) <= idx {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	s := b.slashings
	if b.sharedFieldReferences[slashings].Refs() > 1 {
		s = b.slashingsVal()
		b.sharedFieldReferences[slashings].MinusRef()
		b.sharedFieldReferences[slashings] = stateutil.NewRef(1)
	}

	s[idx] = val

	b.slashings = s

	b.markFieldAsDirty(slashings)
	return nil
}

// AppendValidator for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendValidator(val *ethpb.Validator) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	vals := b.validators
	if b.sharedFieldReferences[validators].Refs() > 1 {
		vals = b.validatorsReferences()
		b.sharedFieldReferences[validators].MinusRef()
		b.sharedFieldReferences[validators] = stateutil.NewRef(1)
	}

	// append validator to slice
	b.validators = append(vals, val)
	valIdx := types.ValidatorIndex(len(b.validators) - 1)

	b.valMapHandler.Set(bytesutil.ToBytes48(val.PublicKey), valIdx)

	b.markFieldAsDirty(validators)
	b.addDirtyIndices(validators, []uint64{uint64(valIdx)})
	return nil
}

// AppendBalance for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendBalance(bal uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	bals := b.balances
	if b.sharedFieldReferences[balances].Refs() > 1 {
		bals = b.balancesVal()
		b.sharedFieldReferences[balances].MinusRef()
		b.sharedFieldReferences[balances] = stateutil.NewRef(1)
	}

	b.balances = append(bals, bal)
	balIdx := len(b.balances) - 1
	b.markFieldAsDirty(balances)
	b.addDirtyIndices(balances, []uint64{uint64(balIdx)})
	return nil
}

// AppendInactivityScore for the beacon state.
func (b *BeaconState) AppendInactivityScore(s uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	scores := b.inactivityScores
	if b.sharedFieldReferences[inactivityScores].Refs() > 1 {
		scores = b.inactivityScoresVal()
		b.sharedFieldReferences[inactivityScores].MinusRef()
		b.sharedFieldReferences[inactivityScores] = stateutil.NewRef(1)
	}

	b.inactivityScores = append(scores, s)
	b.markFieldAsDirty(inactivityScores)
	return nil
}

// SetInactivityScores for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetInactivityScores(val []uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[inactivityScores].MinusRef()
	b.sharedFieldReferences[inactivityScores] = stateutil.NewRef(1)

	b.inactivityScores = val
	b.markFieldAsDirty(inactivityScores)
	return nil
}

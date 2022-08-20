package state_native

import (
	"github.com/pkg/errors"
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// SetValidators for the beacon state. Updates the entire
// to a new value by overwriting the previous one.
func (b *BeaconState) SetValidators(val []*ethpb.Validator) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.validators = val
	b.sharedFieldReferences[nativetypes.Validators].MinusRef()
	b.sharedFieldReferences[nativetypes.Validators] = stateutil.NewRef(1)
	b.markFieldAsDirty(nativetypes.Validators)
	b.rebuildTrie[nativetypes.Validators] = true
	b.valMapHandler = stateutil.NewValMapHandler(b.validators)
	return nil
}

// ApplyToEveryValidator applies the provided callback function to each validator in the
// validator registry.
func (b *BeaconState) ApplyToEveryValidator(f func(idx int, val *ethpb.Validator) (bool, *ethpb.Validator, error)) error {
	b.lock.Lock()
	v := b.validators
	if ref := b.sharedFieldReferences[nativetypes.Validators]; ref.Refs() > 1 {
		v = b.validatorsReferences()
		ref.MinusRef()
		b.sharedFieldReferences[nativetypes.Validators] = stateutil.NewRef(1)
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
	b.markFieldAsDirty(nativetypes.Validators)
	b.addDirtyIndices(nativetypes.Validators, changedVals)

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
	if ref := b.sharedFieldReferences[nativetypes.Validators]; ref.Refs() > 1 {
		v = b.validatorsReferences()
		ref.MinusRef()
		b.sharedFieldReferences[nativetypes.Validators] = stateutil.NewRef(1)
	}

	v[idx] = val
	b.validators = v
	b.markFieldAsDirty(nativetypes.Validators)
	b.addDirtyIndices(nativetypes.Validators, []uint64{uint64(idx)})

	return nil
}

// SetBalances for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetBalances(val []uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[nativetypes.Balances].MinusRef()
	b.sharedFieldReferences[nativetypes.Balances] = stateutil.NewRef(1)

	b.balances = val
	b.markFieldAsDirty(nativetypes.Balances)
	b.rebuildTrie[nativetypes.Balances] = true
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
	if b.sharedFieldReferences[nativetypes.Balances].Refs() > 1 {
		bals = b.balancesVal()
		b.sharedFieldReferences[nativetypes.Balances].MinusRef()
		b.sharedFieldReferences[nativetypes.Balances] = stateutil.NewRef(1)
	}

	bals[idx] = val
	b.balances = bals
	b.markFieldAsDirty(nativetypes.Balances)
	b.addDirtyIndices(nativetypes.Balances, []uint64{uint64(idx)})
	return nil
}

// SetSlashings for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetSlashings(val []uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[nativetypes.Slashings].MinusRef()
	b.sharedFieldReferences[nativetypes.Slashings] = stateutil.NewRef(1)

	b.slashings = val
	b.markFieldAsDirty(nativetypes.Slashings)
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
	if b.sharedFieldReferences[nativetypes.Slashings].Refs() > 1 {
		s = b.slashingsVal()
		b.sharedFieldReferences[nativetypes.Slashings].MinusRef()
		b.sharedFieldReferences[nativetypes.Slashings] = stateutil.NewRef(1)
	}

	s[idx] = val

	b.slashings = s

	b.markFieldAsDirty(nativetypes.Slashings)
	return nil
}

// AppendValidator for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendValidator(val *ethpb.Validator) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	vals := b.validators
	if b.sharedFieldReferences[nativetypes.Validators].Refs() > 1 {
		vals = b.validatorsReferences()
		b.sharedFieldReferences[nativetypes.Validators].MinusRef()
		b.sharedFieldReferences[nativetypes.Validators] = stateutil.NewRef(1)
	}

	// append validator to slice
	b.validators = append(vals, val)
	valIdx := types.ValidatorIndex(len(b.validators) - 1)

	b.valMapHandler.Set(bytesutil.ToBytes48(val.PublicKey), valIdx)

	b.markFieldAsDirty(nativetypes.Validators)
	b.addDirtyIndices(nativetypes.Validators, []uint64{uint64(valIdx)})
	return nil
}

// AppendBalance for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendBalance(bal uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	bals := b.balances
	if b.sharedFieldReferences[nativetypes.Balances].Refs() > 1 {
		bals = b.balancesVal()
		b.sharedFieldReferences[nativetypes.Balances].MinusRef()
		b.sharedFieldReferences[nativetypes.Balances] = stateutil.NewRef(1)
	}

	b.balances = append(bals, bal)
	balIdx := len(b.balances) - 1
	b.markFieldAsDirty(nativetypes.Balances)
	b.addDirtyIndices(nativetypes.Balances, []uint64{uint64(balIdx)})
	return nil
}

// AppendInactivityScore for the beacon state.
func (b *BeaconState) AppendInactivityScore(s uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return errNotSupported("AppendInactivityScore", b.version)
	}

	scores := b.inactivityScores
	if b.sharedFieldReferences[nativetypes.InactivityScores].Refs() > 1 {
		scores = b.inactivityScoresVal()
		b.sharedFieldReferences[nativetypes.InactivityScores].MinusRef()
		b.sharedFieldReferences[nativetypes.InactivityScores] = stateutil.NewRef(1)
	}

	b.inactivityScores = append(scores, s)
	b.markFieldAsDirty(nativetypes.InactivityScores)
	return nil
}

// SetInactivityScores for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetInactivityScores(val []uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return errNotSupported("SetInactivityScores", b.version)
	}

	b.sharedFieldReferences[nativetypes.InactivityScores].MinusRef()
	b.sharedFieldReferences[nativetypes.InactivityScores] = stateutil.NewRef(1)

	b.inactivityScores = val
	b.markFieldAsDirty(nativetypes.InactivityScores)
	return nil
}

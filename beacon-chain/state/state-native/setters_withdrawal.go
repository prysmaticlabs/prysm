package state_native

import (
	"fmt"

	"github.com/pkg/errors"
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// SetWithdrawalQueue for the beacon state. Updates the entire list
// to a new value by overwriting the previous one.
func (b *BeaconState) SetWithdrawalQueue(val []*enginev1.Withdrawal) error {
	if b.version < version.Capella {
		return errNotSupported("SetWithdrawalQueue", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.withdrawalQueue = val
	b.sharedFieldReferences[nativetypes.WithdrawalQueue].MinusRef()
	b.sharedFieldReferences[nativetypes.WithdrawalQueue] = stateutil.NewRef(1)
	b.markFieldAsDirty(nativetypes.WithdrawalQueue)
	b.rebuildTrie[nativetypes.WithdrawalQueue] = true
	return nil
}

// AppendWithdrawal adds a new withdrawal to the end of withdrawal queue.
func (b *BeaconState) AppendWithdrawal(val *enginev1.Withdrawal) error {
	if b.version < version.Capella {
		return errNotSupported("AppendWithdrawal", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	q := b.withdrawalQueue
	max := uint64(fieldparams.ValidatorRegistryLimit)
	if uint64(len(q)) == max {
		return fmt.Errorf("withdrawal queue has max length %d", max)
	}

	if b.sharedFieldReferences[nativetypes.WithdrawalQueue].Refs() > 1 {
		// Copy elements in underlying array by reference.
		q = make([]*enginev1.Withdrawal, len(b.withdrawalQueue))
		copy(q, b.withdrawalQueue)
		b.sharedFieldReferences[nativetypes.WithdrawalQueue].MinusRef()
		b.sharedFieldReferences[nativetypes.WithdrawalQueue] = stateutil.NewRef(1)
	}

	b.withdrawalQueue = append(q, val)
	b.markFieldAsDirty(nativetypes.WithdrawalQueue)
	b.addDirtyIndices(nativetypes.WithdrawalQueue, []uint64{uint64(len(b.withdrawalQueue) - 1)})
	return nil
}

// SetNextWithdrawalIndex sets the index that will be assigned to the next withdrawal.
func (b *BeaconState) SetNextWithdrawalIndex(i uint64) error {
	if b.version < version.Capella {
		return errNotSupported("SetNextWithdrawalIndex", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.nextWithdrawalIndex = i
	return nil
}

// IncreaseNextWithdrawalIndex increases the index that will be assigned to the next withdrawal.
func (b *BeaconState) IncreaseNextWithdrawalIndex() error {
	if b.version < version.Capella {
		return errNotSupported("IncreaseNextWithdrawalIndex", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.nextWithdrawalIndex += 1
	return nil
}

// SetNextPartialWithdrawalValidatorIndex sets the index of the validator which is
// next in line for a partial withdrawal.
func (b *BeaconState) SetNextPartialWithdrawalValidatorIndex(i types.ValidatorIndex) error {
	if b.version < version.Capella {
		return errNotSupported("SetNextPartialWithdrawalValidatorIndex", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.nextPartialWithdrawalValidatorIndex = i
	return nil
}

// WithdrawBalance withdraws the balance from the validator and creates and
// withdrawal receipt for the EL to process
func (b *BeaconState) WithdrawBalance(index types.ValidatorIndex, amount uint64) error {
	if b.version < version.Capella {
		return errNotSupported("WithdrawBalance", b.version)
	}
	balAtIdx, err := b.BalanceAtIndex(index)
	if err != nil {
		return errors.Wrapf(err, "could not get balance at index %d", index)
	}
	if amount > balAtIdx {
		balAtIdx = 0
	} else {
		balAtIdx -= amount
	}

	if err := b.UpdateBalancesAtIndex(index, balAtIdx); err != nil {
		return errors.Wrapf(err, "could not update balance of validator index %d", index)
	}
	nextWithdrawalIndex, err := b.NextWithdrawalIndex()
	if err != nil {
		return errors.Wrap(err, "could not get the next withdrawal index")
	}
	val, err := b.ValidatorAtIndexReadOnly(index)
	if err != nil {
		return errors.Wrapf(err, "could not get validator at index %d", index)
	}

	// Protection against withdrawing a BLS validator, this should not
	// happen in runtime!
	if !val.HasETH1WithdrawalCredential() {
		return errors.New("could not withdraw balance from validator: invalid withdrawal credentials")
	}

	withdrawal := &enginev1.Withdrawal{
		WithdrawalIndex:  nextWithdrawalIndex,
		ValidatorIndex:   index,
		ExecutionAddress: val.WithdrawalCredentials()[12:],
		Amount:           amount,
	}

	if err := b.IncreaseNextWithdrawalIndex(); err != nil {
		return errors.Wrap(err, "could not increase next withdrawal index")
	}
	return b.AppendWithdrawal(withdrawal)
}

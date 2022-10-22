package state_native

import (
	"fmt"

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

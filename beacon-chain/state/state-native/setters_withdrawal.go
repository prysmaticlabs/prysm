package state_native

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// SetNextWithdrawalIndex sets the index that will be assigned to the next withdrawal.
func (b *BeaconState) SetNextWithdrawalIndex(i uint64) error {
	if b.version < version.Capella {
		return errNotSupported("SetNextWithdrawalIndex", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.nextWithdrawalIndex = i
	b.markFieldAsDirty(types.NextWithdrawalIndex)
	return nil
}

// SetNextWithdrawalValidatorIndex sets the index of the validator which is
// next in line for a partial withdrawal.
func (b *BeaconState) SetNextWithdrawalValidatorIndex(i primitives.ValidatorIndex) error {
	if b.version < version.Capella {
		return errNotSupported("SetNextWithdrawalValidatorIndex", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.nextWithdrawalValidatorIndex = i
	b.markFieldAsDirty(types.NextWithdrawalValidatorIndex)
	return nil
}

func (b *BeaconState) AppendPendingPartialWithdrawal(ppw *eth.PendingPartialWithdrawal) error {
	if b.version < version.Electra {
		return errNotSupported("AppendPendingPartialWithdrawal", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[types.PendingPartialWithdrawals].MinusRef()
	b.sharedFieldReferences[types.PendingPartialWithdrawals] = stateutil.NewRef(1)

	b.pendingPartialWithdrawals = append(b.pendingPartialWithdrawals, ppw)

	b.markFieldAsDirty(types.PendingPartialWithdrawals)
	return nil
}

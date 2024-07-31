package state_native

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// AppendPendingBalanceDeposit is a mutating call to the beacon state to create and append a pending
// balance deposit object on to the state. This method requires access to the Lock on the state and
// only applies in electra or later.
func (b *BeaconState) AppendPendingBalanceDeposit(index primitives.ValidatorIndex, amount uint64) error {
	if b.version < version.Electra {
		return errNotSupported("AppendPendingBalanceDeposit", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[types.PendingBalanceDeposits].MinusRef()
	b.sharedFieldReferences[types.PendingBalanceDeposits] = stateutil.NewRef(1)

	b.pendingBalanceDeposits = append(b.pendingBalanceDeposits, &ethpb.PendingBalanceDeposit{Index: index, Amount: amount})

	b.markFieldAsDirty(types.PendingBalanceDeposits)
	b.rebuildTrie[types.PendingBalanceDeposits] = true
	return nil
}

// SetPendingBalanceDeposits is a mutating call to the beacon state which replaces the pending
// balance deposit slice with the provided value. This method requires access to the Lock on the
// state and only applies in electra or later.
func (b *BeaconState) SetPendingBalanceDeposits(val []*ethpb.PendingBalanceDeposit) error {
	if b.version < version.Electra {
		return errNotSupported("SetPendingBalanceDeposits", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[types.PendingBalanceDeposits].MinusRef()
	b.sharedFieldReferences[types.PendingBalanceDeposits] = stateutil.NewRef(1)

	b.pendingBalanceDeposits = val

	b.markFieldAsDirty(types.PendingBalanceDeposits)
	b.rebuildTrie[types.PendingBalanceDeposits] = true
	return nil
}

// SetDepositBalanceToConsume is a mutating call to the beacon state which sets the deposit balance
// to consume value to the given value. This method requires access to the Lock on the state and
// only applies in electra or later.
func (b *BeaconState) SetDepositBalanceToConsume(dbtc primitives.Gwei) error {
	if b.version < version.Electra {
		return errNotSupported("SetDepositBalanceToConsume", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.depositBalanceToConsume = dbtc

	b.markFieldAsDirty(types.DepositBalanceToConsume)
	b.rebuildTrie[types.DepositBalanceToConsume] = true
	return nil
}

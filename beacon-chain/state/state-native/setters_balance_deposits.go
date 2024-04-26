package state_native

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

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

func (b *BeaconState) SetPendingBalanceDeposits(val []*ethpb.PendingBalanceDeposit) error {
	if b.version < version.Electra {
		return errNotSupported("SetPendingBalanceDeposits", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.pendingBalanceDeposits = val

	b.markFieldAsDirty(types.PendingBalanceDeposits)
	b.rebuildTrie[types.PendingBalanceDeposits] = true
	return nil
}

func (b *BeaconState) SetDepositBalanceToConsume(gwei uint64) error {
	if b.version < version.Electra {
		return errNotSupported("SetDepositBalanceToConsume", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.depositBalanceToConsume = gwei

	b.markFieldAsDirty(types.DepositBalanceToConsume)
	b.rebuildTrie[types.DepositBalanceToConsume] = true
	return nil
}

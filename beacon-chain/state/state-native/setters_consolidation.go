package state_native

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func (b *BeaconState) AppendPendingConsolidation(val *ethpb.PendingConsolidation) error {
	if b.version < version.Electra {
		return errNotSupported("AppendPendingConsolidation", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[types.PendingConsolidations].MinusRef()
	b.sharedFieldReferences[types.PendingConsolidations] = stateutil.NewRef(1)

	b.pendingConsolidations = append(b.pendingConsolidations, val)

	b.markFieldAsDirty(types.PendingConsolidations)
	b.rebuildTrie[types.PendingConsolidations] = true
	return nil
}

func (b *BeaconState) SetPendingConsolidations(val []*ethpb.PendingConsolidation) error {
	if b.version < version.Electra {
		return errNotSupported("SetPendingConsolidations", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[types.PendingConsolidations].MinusRef()
	b.sharedFieldReferences[types.PendingConsolidations] = stateutil.NewRef(1)

	b.pendingConsolidations = val

	b.markFieldAsDirty(types.PendingConsolidations)
	b.rebuildTrie[types.PendingConsolidations] = true
	return nil
}

func (b *BeaconState) SetEarliestConsolidationEpoch(epoch primitives.Epoch) error {
	if b.version < version.Electra {
		return errNotSupported("SetEarliestConsolidationEpoch", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.earliestConsolidationEpoch = epoch

	b.markFieldAsDirty(types.EarliestConsolidationEpoch)
	b.rebuildTrie[types.EarliestConsolidationEpoch] = true
	return nil
}

func (b *BeaconState) SetConsolidationBalanceToConsume(balance uint64) error {
	if b.version < version.Electra {
		return errNotSupported("SetConsolidationBalanceToConsume", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.consolidationBalanceToConsume = balance

	b.markFieldAsDirty(types.ConsolidationBalanceToConsume)
	b.rebuildTrie[types.ConsolidationBalanceToConsume] = true
	return nil
}

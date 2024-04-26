package state_native

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func (b *BeaconState) EarliestConsolidationEpoch() (primitives.Epoch, error) {
	if b.version < version.Electra {
		return 0, errNotSupported("EarliestConsolidationEpoch", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.earliestConsolidationEpoch, nil
}

func (b *BeaconState) ConsolidationBalanceToConsume() (uint64, error) {
	if b.version < version.Electra {
		return 0, errNotSupported("ConsolidationBalanceToConsume", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.consolidationBalanceToConsume, nil
}

func (b *BeaconState) PendingConsolidations() ([]*ethpb.PendingConsolidation, error) {
	if b.version < version.Electra {
		return nil, errNotSupported("PendingConsolidations", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.pendingConsolidationsVal(), nil
}

func (b *BeaconState) NumPendingConsolidations() uint64 {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return uint64(len(b.pendingConsolidations))
}

func (b *BeaconState) pendingConsolidationsVal() []*ethpb.PendingConsolidation {
	if b.pendingConsolidations == nil {
		return nil
	}

	return ethpb.CopyPendingConsolidations(b.pendingConsolidations)
}

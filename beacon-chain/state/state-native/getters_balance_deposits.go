package state_native

import (
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func (b *BeaconState) DepositBalanceToConsume() (uint64, error) {
	if b.version < version.Electra {
		return 0, errNotSupported("DepositBalanceToConsume", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.depositBalanceToConsume, nil
}

func (b *BeaconState) PendingBalanceDeposits() ([]*ethpb.PendingBalanceDeposit, error) {
	if b.version < version.Electra {
		return nil, errNotSupported("PendingBalanceDeposits", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.pendingBalanceDepositsVal(), nil
}

func (b *BeaconState) pendingBalanceDepositsVal() []*ethpb.PendingBalanceDeposit {
	if b.pendingBalanceDeposits == nil {
		return nil
	}

	return ethpb.CopyPendingBalanceDeposits(b.pendingBalanceDeposits)
}

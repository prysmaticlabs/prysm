package state_native

import (
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// WithdrawalQueue returns the list of pending withdrawals.
func (b *BeaconState) WithdrawalQueue() ([]*enginev1.Withdrawal, error) {
	if b.version < version.Capella {
		return nil, errNotSupported("WithdrawalQueue", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.withdrawalQueueVal(), nil
}

func (b *BeaconState) withdrawalQueueVal() []*enginev1.Withdrawal {
	return ethpb.CopyWithdrawalSlice(b.withdrawalQueue)
}

// NextWithdrawalIndex returns the index that will be assigned to the next withdrawal.
func (b *BeaconState) NextWithdrawalIndex() (uint64, error) {
	if b.version < version.Capella {
		return 0, errNotSupported("NextWithdrawalIndex", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.nextWithdrawalIndex, nil
}

// NextPartialWithdrawalValidatorIndex returns the index of the validator which is
// next in line for a partial withdrawal.
func (b *BeaconState) NextPartialWithdrawalValidatorIndex() (types.ValidatorIndex, error) {
	if b.version < version.Capella {
		return 0, errNotSupported("NextPartialWithdrawalValidatorIndex", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.nextPartialWithdrawalValidatorIndex, nil
}

package state_native

import (
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

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

// SetLastWithdrawalValidatorIndex sets the index of the validator which is
// next in line for a partial withdrawal.
func (b *BeaconState) SetLastWithdrawalValidatorIndex(i types.ValidatorIndex) error {
	if b.version < version.Capella {
		return errNotSupported("SetNextPartialWithdrawalValidatorIndex", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.lastWithdrawalValidatorIndex = i
	return nil
}

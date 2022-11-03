package state_native

import (
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

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
func (b *BeaconState) LastWithdrawalValidatorIndex() (types.ValidatorIndex, error) {
	if b.version < version.Capella {
		return 0, errNotSupported("LastWithdrawalValidatorIndex", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.lastWithdrawalValidatorIndex, nil
}

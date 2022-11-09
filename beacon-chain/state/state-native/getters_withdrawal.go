package state_native

import (
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

const ETH1AddressOffset = 12

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

// ExpectedWithdrawals returns the withdrawals that a proposer will need to pack in the next block
// applied to the current state. It is also used by validators to check that the execution payload carried
// the right number of withdrawals
func (b *BeaconState) ExpectedWithdrawals() ([]*enginev1.Withdrawal, error) {
	if b.version < version.Capella {
		return nil, errNotSupported("ExpectedWithdrawals", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	withdrawals := make([]*enginev1.Withdrawal, 0, params.BeaconConfig().MaxWithdrawalsPerPayload)
	validatorIndex := b.lastWithdrawalValidatorIndex
	withdrawalIndex := b.nextWithdrawalIndex
	epoch := slots.ToEpoch(b.slot)
	for range b.validators {
		validatorIndex += 1
		if uint64(validatorIndex) == uint64(len(b.validators)) {
			validatorIndex = types.ValidatorIndex(0)
		}
		val := b.validators[validatorIndex]
		balance := b.balances[validatorIndex]
		if isFullyWithdrawableValidator(val, epoch) {
			withdrawals = append(withdrawals, &enginev1.Withdrawal{
				WithdrawalIndex:  withdrawalIndex,
				ValidatorIndex:   validatorIndex,
				ExecutionAddress: bytesutil.SafeCopyBytes(val.WithdrawalCredentials[ETH1AddressOffset:]),
				Amount:           balance,
			})
			withdrawalIndex++
		} else if isPartiallyWithdrawableValidator(val, balance) {
			withdrawals = append(withdrawals, &enginev1.Withdrawal{
				WithdrawalIndex:  withdrawalIndex,
				ValidatorIndex:   validatorIndex,
				ExecutionAddress: bytesutil.SafeCopyBytes(val.WithdrawalCredentials[ETH1AddressOffset:]),
				Amount:           balance - params.BeaconConfig().MaxEffectiveBalance,
			})
			withdrawalIndex++
		}
		if uint64(len(withdrawals)) == params.BeaconConfig().MaxWithdrawalsPerPayload {
			break
		}
	}
	return withdrawals, nil
}

// hasETH1WithdrawalCredential returns whether the validator has an ETH1
// Withdrawal prefix. It assumes that the caller has a lock on the state
func hasETH1WithdrawalCredential(val *ethpb.Validator) bool {
	if val == nil {
		return false
	}
	cred := val.WithdrawalCredentials
	return len(cred) > 0 && cred[0] == params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
}

// isFullyWithdrawableValidator returns whether the validator is able to perform a full
// withdrawal. This differ from the spec helper in that the balance > 0 is not
// checked. This function assumes that the caller holds a lock on the state
func isFullyWithdrawableValidator(val *ethpb.Validator, epoch types.Epoch) bool {
	if val == nil {
		return false
	}
	return hasETH1WithdrawalCredential(val) && val.WithdrawableEpoch <= epoch
}

// isPartiallyWithdrawable returns whether the validator is able to perform a
// partial withdrawal. This function assumes that the caller has a lock on the state
func isPartiallyWithdrawableValidator(val *ethpb.Validator, balance uint64) bool {
	if val == nil {
		return false
	}
	hasMaxBalance := val.EffectiveBalance == params.BeaconConfig().MaxEffectiveBalance
	hasExcessBalance := balance > params.BeaconConfig().MaxEffectiveBalance
	return hasETH1WithdrawalCredential(val) && hasExcessBalance && hasMaxBalance
}

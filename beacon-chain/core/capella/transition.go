package capella

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
)

// ETH1AddressOffset is the offset at which the execution address starts
// within an ETH1-prefixed withdrawal credential
const ETH1AddressOffset = 12

// withdrawBalance withdraws the balance from the validator and creates a
// withdrawal receipt for the EL to process
func withdrawBalance(pre state.BeaconState, index types.ValidatorIndex, amount uint64) (state.BeaconState, error) {
	b := pre.Copy()
	val, err := b.ValidatorAtIndexReadOnly(index)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get validator at index %d", index)
	}

	// Protection against withdrawing a BLS validator, this should not
	// happen in runtime!
	if !val.HasETH1WithdrawalCredential() {
		return nil, errors.New("could not withdraw balance from validator: invalid withdrawal credentials")
	}

	if err := helpers.DecreaseBalance(b, index, amount); err != nil {
		return nil, errors.Wrap(err, "could not decrease balance")
	}

	withdrawalIndex, err := b.NextWithdrawalIndex()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next withdrawal index")
	}

	withdrawal := &enginev1.Withdrawal{
		WithdrawalIndex:  withdrawalIndex,
		ValidatorIndex:   index,
		ExecutionAddress: val.WithdrawalCredentials()[ETH1AddressOffset:],
		Amount:           amount,
	}

	if err := b.IncreaseNextWithdrawalIndex(); err != nil {
		return nil, errors.Wrap(err, "could not increase next withdrawal index")
	}

	if err := b.AppendWithdrawal(withdrawal); err != nil {
		return nil, errors.Wrap(err, "could not append withdrawal")
	}
	return b, nil
}

// processWithdrawalsIntoQueue process the withdrawals at epoch transition, it
// sweeps through the validator set and queues the withdrawals of up to
// `MAX_WITHDRAWALS_PER_EPOCH`.
func processWithdrawalsIntoQueue(pre state.BeaconState) (state.BeaconState, error) {
	s := pre.Copy()
	validators := s.Validators()
	index, err := s.NextWithdrawalValidatorIndex()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next withdrawal index")
	}
	// protection for invalid next withdrawal validator index
	// This should not happen in runtime
	if index >= types.ValidatorIndex(len(validators)) {
		return nil, errors.New("could not process withdrawals: invalid next withdrawal index")
	}

	epoch := time.CurrentEpoch(s)
	for count := uint64(0); count < params.BeaconConfig().MaxWithdrawalsPerEpoch; {
		val, err := s.ValidatorAtIndexReadOnly(index)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get validator at index %d", index)
		}
		balance, err := s.BalanceAtIndex(index)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get balance of validator at index %d", index)
		}
		if val.IsFullyWithdrawable(epoch) {
			s, err = withdrawBalance(s, index, balance)
			if err != nil {
				return nil, errors.Wrap(err, "could not withdraw validator balance")
			}
			count++
		} else if val.IsPartiallyWithdrawable(balance) {
			s, err = withdrawBalance(s, index, balance-params.BeaconConfig().MaxEffectiveBalance)
			if err != nil {
				return nil, errors.Wrap(err, "could not withdraw validator balance")
			}
			count++
		} else {
			index++
			if index == types.ValidatorIndex(len(validators)) {
				index = types.ValidatorIndex(0)
			}
		}
	}
	return s, nil
}

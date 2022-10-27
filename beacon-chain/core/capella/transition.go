package capella

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
)

// WithdrawBalance withdraws the balance from the validator and creates a
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
		ExecutionAddress: val.WithdrawalCredentials()[12:],
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

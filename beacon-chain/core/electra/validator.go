package electra

import (
	"errors"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// AddValidatorToRegistry updates the beacon state with validator information
// def add_validator_to_registry(state: BeaconState,
//
//	                          pubkey: BLSPubkey,
//	                          withdrawal_credentials: Bytes32,
//	                          amount: uint64) -> None:
//	index = get_index_for_new_validator(state)
//	validator = get_validator_from_deposit(pubkey, withdrawal_credentials)
//	set_or_append_list(state.validators, index, validator)
//	set_or_append_list(state.balances, index, 0)  # [Modified in Electra:EIP7251]
//	set_or_append_list(state.previous_epoch_participation, index, ParticipationFlags(0b0000_0000))
//	set_or_append_list(state.current_epoch_participation, index, ParticipationFlags(0b0000_0000))
//	set_or_append_list(state.inactivity_scores, index, uint64(0))
//	state.pending_balance_deposits.append(PendingBalanceDeposit(index=index, amount=amount))  # [New in Electra:EIP7251]
func AddValidatorToRegistry(beaconState state.BeaconState, pubKey []byte, withdrawalCredentials []byte, amount uint64) error {
	val := ValidatorFromDeposit(pubKey, withdrawalCredentials)
	if err := beaconState.AppendValidator(val); err != nil {
		return err
	}
	index, ok := beaconState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
	if !ok {
		return errors.New("could not find validator in registry")
	}
	if err := beaconState.AppendBalance(0); err != nil {
		return err
	}
	if err := beaconState.AppendPendingBalanceDeposit(index, amount); err != nil {
		return err
	}
	if err := beaconState.AppendInactivityScore(0); err != nil {
		return err
	}
	if err := beaconState.AppendPreviousParticipationBits(0); err != nil {
		return err
	}
	return beaconState.AppendCurrentParticipationBits(0)
}

// ValidatorFromDeposit gets a new validator object with provided parameters
//
// def get_validator_from_deposit(pubkey: BLSPubkey, withdrawal_credentials: Bytes32) -> Validator:
//
//	return Validator(
//	pubkey=pubkey,
//	withdrawal_credentials=withdrawal_credentials,
//	activation_eligibility_epoch=FAR_FUTURE_EPOCH,
//	activation_epoch=FAR_FUTURE_EPOCH,
//	exit_epoch=FAR_FUTURE_EPOCH,
//	withdrawable_epoch=FAR_FUTURE_EPOCH,
//	effective_balance=0,  # [Modified in Electra:EIP7251]
//
// )
func ValidatorFromDeposit(pubKey []byte, withdrawalCredentials []byte) *ethpb.Validator {
	return &ethpb.Validator{
		PublicKey:                  pubKey,
		WithdrawalCredentials:      withdrawalCredentials,
		ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
		ActivationEpoch:            params.BeaconConfig().FarFutureEpoch,
		ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
		WithdrawableEpoch:          params.BeaconConfig().FarFutureEpoch,
		EffectiveBalance:           0, // [Modified in Electra:EIP7251]
	}
}

// SwitchToCompoundingValidator
//
// Spec definition:
//
//	 def switch_to_compounding_validator(state: BeaconState, index: ValidatorIndex) -> None:
//		validator = state.validators[index]
//		if has_eth1_withdrawal_credential(validator):
//		    validator.withdrawal_credentials = COMPOUNDING_WITHDRAWAL_PREFIX + validator.withdrawal_credentials[1:]
//		    queue_excess_active_balance(state, index)
func SwitchToCompoundingValidator(s state.BeaconState, idx primitives.ValidatorIndex) error {
	v, err := s.ValidatorAtIndex(idx)
	if err != nil {
		return err
	}
	if len(v.WithdrawalCredentials) == 0 {
		return errors.New("validator has no withdrawal credentials")
	}
	if helpers.HasETH1WithdrawalCredential(v) {
		v.WithdrawalCredentials[0] = params.BeaconConfig().CompoundingWithdrawalPrefixByte
		if err := s.UpdateValidatorAtIndex(idx, v); err != nil {
			return err
		}
		return QueueExcessActiveBalance(s, idx)
	}
	return nil
}

// QueueExcessActiveBalance queues validators with balances above the min activation balance and adds to pending balance deposit.
//
// Spec definition:
//
//	def queue_excess_active_balance(state: BeaconState, index: ValidatorIndex) -> None:
//	    balance = state.balances[index]
//	    if balance > MIN_ACTIVATION_BALANCE:
//	        excess_balance = balance - MIN_ACTIVATION_BALANCE
//	        state.balances[index] = MIN_ACTIVATION_BALANCE
//	        state.pending_balance_deposits.append(
//	            PendingBalanceDeposit(index=index, amount=excess_balance)
//	        )
func QueueExcessActiveBalance(s state.BeaconState, idx primitives.ValidatorIndex) error {
	bal, err := s.BalanceAtIndex(idx)
	if err != nil {
		return err
	}

	if bal > params.BeaconConfig().MinActivationBalance {
		excessBalance := bal - params.BeaconConfig().MinActivationBalance
		if err := s.UpdateBalancesAtIndex(idx, params.BeaconConfig().MinActivationBalance); err != nil {
			return err
		}
		return s.AppendPendingBalanceDeposit(idx, excessBalance)
	}
	return nil
}

// QueueEntireBalanceAndResetValidator queues the entire balance and resets the validator. This is used in electra fork logic.
//
// Spec definition:
//
//	def queue_entire_balance_and_reset_validator(state: BeaconState, index: ValidatorIndex) -> None:
//	    balance = state.balances[index]
//	    state.balances[index] = 0
//	    validator = state.validators[index]
//	    validator.effective_balance = 0
//	    validator.activation_eligibility_epoch = FAR_FUTURE_EPOCH
//	    state.pending_balance_deposits.append(
//	        PendingBalanceDeposit(index=index, amount=balance)
//	    )
//
//nolint:dupword
func QueueEntireBalanceAndResetValidator(s state.BeaconState, idx primitives.ValidatorIndex) error {
	bal, err := s.BalanceAtIndex(idx)
	if err != nil {
		return err
	}

	if err := s.UpdateBalancesAtIndex(idx, 0); err != nil {
		return err
	}

	v, err := s.ValidatorAtIndex(idx)
	if err != nil {
		return err
	}

	v.EffectiveBalance = 0
	v.ActivationEligibilityEpoch = params.BeaconConfig().FarFutureEpoch
	if err := s.UpdateValidatorAtIndex(idx, v); err != nil {
		return err
	}

	return s.AppendPendingBalanceDeposit(idx, bal)
}

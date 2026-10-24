package eip8453

import (
	"context"
	"fmt"
	"math/bits"
	"slices"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/electra"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/time"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/pkg/errors"
)

// ProcessBuilderPendingPayments pays the previous epoch's builder payments whose slot reached the head vote seat quorum.
func ProcessBuilderPendingPayments(st state.BeaconState) error {
	payments, err := st.BuilderPendingPayments()
	if err != nil {
		return err
	}
	participation, err := st.HeadVoteParticipation()
	if err != nil {
		return err
	}
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	var withdrawals []*ethpb.BuilderPendingWithdrawal
	for index := range slotsPerEpoch {
		payment := payments[index]
		if payment.Withdrawal.Amount == 0 {
			continue
		}
		if hasBuilderPaymentQuorum(participation[index]) {
			withdrawals = append(withdrawals, payment.Withdrawal)
		}
	}
	if err := st.AppendBuilderPendingWithdrawals(withdrawals); err != nil {
		return err
	}
	if err := st.RotateBuilderPendingPayments(); err != nil {
		return err
	}
	rotated := make([][]byte, len(participation))
	copy(rotated, participation[slotsPerEpoch:])
	for index := slotsPerEpoch; index < uint64(len(rotated)); index++ {
		rotated[index] = make([]byte, len(participation[index]))
	}
	return st.SetHeadVoteParticipation(rotated)
}

func hasBuilderPaymentQuorum(seats []byte) bool {
	var count uint64
	for _, b := range seats {
		count += uint64(bits.OnesCount8(b))
	}
	cfg := params.BeaconConfig()
	return count*cfg.BuilderPaymentThresholdDenominator >= cfg.HeadVoteCommitteeSize*cfg.BuilderPaymentThresholdNumerator
}

// ProcessPendingDeposits applies queued deposits whose slot is finalized, within the activation churn.
func ProcessPendingDeposits(ctx context.Context, st state.BeaconState) error {
	nextEpoch := time.CurrentEpoch(st) + 1
	toConsume, err := st.DepositBalanceToConsume()
	if err != nil {
		return err
	}
	total, err := helpers.TotalActiveBalance(ctx, st)
	if err != nil {
		return err
	}
	available := toConsume + helpers.ActivationChurnLimitForVersion(st.Version(), primitives.Gwei(total))
	finalizedSlot, err := st.FinalizedSlot()
	if err != nil {
		return err
	}
	deposits, err := st.PendingDeposits()
	if err != nil {
		return err
	}
	var processed primitives.Gwei
	var next uint64
	var postponed []*ethpb.PendingDeposit
	churnLimitReached := false
	for _, deposit := range deposits {
		if deposit.Slot > finalizedSlot {
			break
		}
		if next >= params.BeaconConfig().MaxPendingDepositsPerEpoch {
			break
		}
		exited, withdrawn := false, false
		if index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(deposit.PublicKey)); ok {
			val, err := st.ValidatorAtIndexReadOnly(index)
			if err != nil {
				return err
			}
			exited = val.ExitEpoch() < params.BeaconConfig().FarFutureEpoch
			withdrawn = val.WithdrawableEpoch() < nextEpoch
		}
		if withdrawn {
			if err := applyPendingDeposit(st, deposit); err != nil {
				return err
			}
		} else if exited {
			postponed = append(postponed, deposit)
		} else {
			churnLimitReached = processed+primitives.Gwei(deposit.Amount) > available
			if churnLimitReached {
				break
			}
			processed += primitives.Gwei(deposit.Amount)
			if err := applyPendingDeposit(st, deposit); err != nil {
				return err
			}
		}
		next++
	}
	if err := st.SetPendingDeposits(append(slices.Clone(deposits[next:]), postponed...)); err != nil {
		return err
	}
	if churnLimitReached {
		return st.SetDepositBalanceToConsume(available - processed)
	}
	return st.SetDepositBalanceToConsume(0)
}

func applyPendingDeposit(st state.BeaconState, deposit *ethpb.PendingDeposit) error {
	if index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(deposit.PublicKey)); ok {
		return helpers.IncreaseBalance(st, index, deposit.Amount)
	}
	// Heze stops creating validators with BLS withdrawal credentials, without a refund.
	if len(deposit.WithdrawalCredentials) > 0 && deposit.WithdrawalCredentials[0] == params.BeaconConfig().BLSWithdrawalPrefixByte {
		return nil
	}
	valid, err := helpers.IsValidDepositSignature(&ethpb.Deposit_Data{
		PublicKey:             bytesutil.SafeCopyBytes(deposit.PublicKey),
		WithdrawalCredentials: bytesutil.SafeCopyBytes(deposit.WithdrawalCredentials),
		Amount:                deposit.Amount,
		Signature:             bytesutil.SafeCopyBytes(deposit.Signature),
	})
	if err != nil || !valid {
		return err
	}
	return addValidatorToRegistry(st, deposit.PublicKey, deposit.WithdrawalCredentials, deposit.Amount)
}

func addValidatorToRegistry(st state.BeaconState, pubkey, withdrawalCredentials []byte, amount uint64) error {
	val, err := electra.GetValidatorFromDeposit(pubkey, withdrawalCredentials, amount)
	if err != nil {
		return errors.Wrap(err, "could not get validator from deposit")
	}
	index := st.NumValidators()
	if err := st.AppendValidator(val); err != nil {
		return err
	}
	if err := st.AppendBalance(amount); err != nil {
		return err
	}
	if err := st.AppendInactivityScore(0); err != nil {
		return err
	}
	round, err := st.RoundParticipation()
	if err != nil {
		return err
	}
	if round, err = setOrAppend(round, index); err != nil {
		return err
	}
	if err := st.SetRoundParticipation(round); err != nil {
		return err
	}
	target, err := st.TargetParticipationEIP8453()
	if err != nil {
		return err
	}
	if target, err = setOrAppend(target, index); err != nil {
		return err
	}
	if err := st.SetTargetParticipationEIP8453(target); err != nil {
		return err
	}
	progress, err := st.ProgressParticipation()
	if err != nil {
		return err
	}
	if progress, err = setOrAppend(progress, index); err != nil {
		return err
	}
	if err := st.SetProgressParticipation(progress); err != nil {
		return err
	}
	pendings, err := st.PendingFinalities()
	if err != nil {
		return err
	}
	grown := make([]*ethpb.PendingFinalityEIP8453, len(pendings))
	for i, pending := range pendings {
		participation, err := setOrAppend(pending.Participation, index)
		if err != nil {
			return err
		}
		grown[i] = &ethpb.PendingFinalityEIP8453{Checkpoint: pending.Checkpoint, Slot: pending.Slot, Participation: participation}
	}
	return st.SetPendingFinalities(grown)
}

func setOrAppend(list []byte, index int) ([]byte, error) {
	if index > len(list) {
		return nil, fmt.Errorf("index %d is past the end of a list of %d", index, len(list))
	}
	list = slices.Clone(list)
	if index == len(list) {
		return append(list, 0), nil
	}
	list[index] = 0
	return list, nil
}

package electra

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// UpgradeToElectra updates inputs a generic state to return the version Electra state.
func UpgradeToElectra(state state.BeaconState) (state.BeaconState, error) {
	epoch := time.CurrentEpoch(state)

	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, err
	}
	nextSyncCommittee, err := state.NextSyncCommittee()
	if err != nil {
		return nil, err
	}
	prevEpochParticipation, err := state.PreviousEpochParticipation()
	if err != nil {
		return nil, err
	}
	currentEpochParticipation, err := state.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	inactivityScores, err := state.InactivityScores()
	if err != nil {
		return nil, err
	}
	payloadHeader, err := state.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	txRoot, err := payloadHeader.TransactionsRoot()
	if err != nil {
		return nil, err
	}
	wdRoot, err := payloadHeader.WithdrawalsRoot()
	if err != nil {
		return nil, err
	}
	wi, err := state.NextWithdrawalIndex()
	if err != nil {
		return nil, err
	}
	vi, err := state.NextWithdrawalValidatorIndex()
	if err != nil {
		return nil, err
	}
	summaries, err := state.HistoricalSummaries()
	if err != nil {
		return nil, err
	}
	historicalRoots, err := state.HistoricalRoots()
	if err != nil {
		return nil, err
	}
	excessBlobGas, err := payloadHeader.ExcessBlobGas()
	if err != nil {
		return nil, err
	}
	blobGasUsed, err := payloadHeader.BlobGasUsed()
	if err != nil {
		return nil, err
	}

	// RTFM: https://github.com/ethereum/consensus-specs/blob/dev/specs/electra/fork.md
	// Find the earliest exit epoch
	exitEpochs := make([]primitives.Epoch, 0)
	for _, v := range state.Validators() {
		if v.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, v.ExitEpoch)
		}
	}
	if len(exitEpochs) == 0 {
		exitEpochs = append(exitEpochs, time.CurrentEpoch(state))
	}
	var earliestExitEpoch primitives.Epoch
	for _, e := range exitEpochs {
		if e > earliestExitEpoch {
			earliestExitEpoch = e
		}
	}
	earliestExitEpoch++ // Increment to find the earliest possible exit epoch

	s := &ethpb.BeaconStateElectra{
		GenesisTime:           state.GenesisTime(),
		GenesisValidatorsRoot: state.GenesisValidatorsRoot(),
		Slot:                  state.Slot(),
		Fork: &ethpb.Fork{
			PreviousVersion: state.Fork().CurrentVersion,
			CurrentVersion:  params.BeaconConfig().ElectraForkVersion,
			Epoch:           epoch,
		},
		LatestBlockHeader:           state.LatestBlockHeader(),
		BlockRoots:                  state.BlockRoots(),
		StateRoots:                  state.StateRoots(),
		HistoricalRoots:             historicalRoots,
		Eth1Data:                    state.Eth1Data(),
		Eth1DataVotes:               state.Eth1DataVotes(),
		Eth1DepositIndex:            state.Eth1DepositIndex(),
		Validators:                  state.Validators(),
		Balances:                    state.Balances(),
		RandaoMixes:                 state.RandaoMixes(),
		Slashings:                   state.Slashings(),
		PreviousEpochParticipation:  prevEpochParticipation,
		CurrentEpochParticipation:   currentEpochParticipation,
		JustificationBits:           state.JustificationBits(),
		PreviousJustifiedCheckpoint: state.PreviousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:  state.CurrentJustifiedCheckpoint(),
		FinalizedCheckpoint:         state.FinalizedCheckpoint(),
		InactivityScores:            inactivityScores,
		CurrentSyncCommittee:        currentSyncCommittee,
		NextSyncCommittee:           nextSyncCommittee,
		LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderElectra{
			ParentHash:             payloadHeader.ParentHash(),
			FeeRecipient:           payloadHeader.FeeRecipient(),
			StateRoot:              payloadHeader.StateRoot(),
			ReceiptsRoot:           payloadHeader.ReceiptsRoot(),
			LogsBloom:              payloadHeader.LogsBloom(),
			PrevRandao:             payloadHeader.PrevRandao(),
			BlockNumber:            payloadHeader.BlockNumber(),
			GasLimit:               payloadHeader.GasLimit(),
			GasUsed:                payloadHeader.GasUsed(),
			Timestamp:              payloadHeader.Timestamp(),
			ExtraData:              payloadHeader.ExtraData(),
			BaseFeePerGas:          payloadHeader.BaseFeePerGas(),
			BlockHash:              payloadHeader.BlockHash(),
			TransactionsRoot:       txRoot,
			WithdrawalsRoot:        wdRoot,
			ExcessBlobGas:          excessBlobGas,
			BlobGasUsed:            blobGasUsed,
			DepositReceiptsRoot:    bytesutil.Bytes32(0),
			WithdrawalRequestsRoot: bytesutil.Bytes32(0),
		},
		NextWithdrawalIndex:          wi,
		NextWithdrawalValidatorIndex: vi,
		HistoricalSummaries:          summaries,

		// TODO: Verify these initial electra values are correct
		// They are not zero!
		DepositReceiptsStartIndex:     0,
		DepositBalanceToConsume:       0,
		ExitBalanceToConsume:          0,
		EarliestExitEpoch:             0,
		ConsolidationBalanceToConsume: 0,
		EarliestConsolidationEpoch:    0,
		PendingBalanceDeposits:        nil,
		PendingPartialWithdrawals:     nil,
		PendingConsolidations:         nil,
	}
	// TODO: more logic to do
	//post.exit_balance_to_consume = get_activation_exit_churn_limit(post)
	//post.consolidation_balance_to_consume = get_consolidation_churn_limit(post)
	//
	//# [New in Electra:EIP7251]
	//# add validators that are not yet active to pending balance deposits
	//pre_activation = sorted([
	//index for index, validator in enumerate(post.validators)
	//if validator.activation_epoch == FAR_FUTURE_EPOCH
	//], key=lambda index: (
	//post.validators[index].activation_eligibility_epoch,
	//index
	//))
	//
	//for index in pre_activation:
	//queue_entire_balance_and_reset_validator(post, ValidatorIndex(index))
	//
	//# Ensure early adopters of compounding credentials go through the activation churn
	//for index, validator in enumerate(post.validators):
	//if has_compounding_withdrawal_credential(validator):
	//queue_excess_active_balance(post, ValidatorIndex(index))

	return state_native.InitializeFromProtoUnsafeElectra(s)
}

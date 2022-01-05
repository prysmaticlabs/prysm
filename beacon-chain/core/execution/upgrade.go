package execution

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state-native"
	statemerge "github.com/prysmaticlabs/prysm/beacon-chain/state-native/v3"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// UpgradeToMerge updates inputs a generic state to return the version Merge state.
// It inserts an empty `ExecutionPayloadHeader` into the state.
func UpgradeToMerge(ctx context.Context, state state.BeaconState) (state.BeaconState, error) {
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

	newState, err := statemerge.Initialize()
	if err != nil {
		return nil, err
	}

	if err = newState.SetGenesisTime(state.GenesisTime()); err != nil {
		return nil, errors.Wrap(err, "could not set genesis time")
	}
	if err = newState.SetGenesisValidatorRoot(state.GenesisValidatorRoot()); err != nil {
		return nil, errors.Wrap(err, "could not set genesis validators root")
	}
	if err = newState.SetSlot(state.Slot()); err != nil {
		return nil, errors.Wrap(err, "could not set slot")
	}
	if err = newState.SetFork(&ethpb.Fork{
		PreviousVersion: state.Fork().CurrentVersion,
		CurrentVersion:  params.BeaconConfig().BellatrixForkVersion,
		Epoch:           epoch,
	}); err != nil {
		return nil, errors.Wrap(err, "could not set fork")
	}
	if err = newState.SetLatestBlockHeader(state.LatestBlockHeader()); err != nil {
		return nil, errors.Wrap(err, "could not set latest block header")
	}
	if err = newState.SetBlockRoots(state.BlockRoots()); err != nil {
		return nil, errors.Wrap(err, "could not set block roots")
	}
	if err = newState.SetStateRoots(state.StateRoots()); err != nil {
		return nil, errors.Wrap(err, "could not set state roots")
	}
	if err = newState.SetHistoricalRoots(state.HistoricalRoots()); err != nil {
		return nil, errors.Wrap(err, "could not set historical roots")
	}
	if err = newState.SetEth1Data(state.Eth1Data()); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data")
	}
	if err = newState.SetEth1DataVotes(state.Eth1DataVotes()); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data votes")
	}
	if err = newState.SetEth1DepositIndex(state.Eth1DepositIndex()); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 deposit index")
	}
	if err = newState.SetValidators(state.Validators()); err != nil {
		return nil, errors.Wrap(err, "could not set validators")
	}
	if err = newState.SetBalances(state.Balances()); err != nil {
		return nil, errors.Wrap(err, "could not set balances")
	}
	if err = newState.SetRandaoMixes(state.RandaoMixes()); err != nil {
		return nil, errors.Wrap(err, "could not set randao mixes")
	}
	if err = newState.SetSlashings(state.Slashings()); err != nil {
		return nil, errors.Wrap(err, "could not set slashings")
	}
	if err = newState.SetJustificationBits(state.JustificationBits()); err != nil {
		return nil, errors.Wrap(err, "could not set justification bits")
	}
	if err = newState.SetPreviousParticipationBits(prevEpochParticipation); err != nil {
		return nil, errors.Wrap(err, "could not set previous participation bits")
	}
	if err = newState.SetCurrentParticipationBits(currentEpochParticipation); err != nil {
		return nil, errors.Wrap(err, "could not set current participation bits")
	}
	if err = newState.SetInactivityScores(inactivityScores); err != nil {
		return nil, errors.Wrap(err, "could not set inactivity scores")
	}
	if err = newState.SetPreviousJustifiedCheckpoint(state.PreviousJustifiedCheckpoint()); err != nil {
		return nil, errors.Wrap(err, "could not set previous justified checkpoint")
	}
	if err = newState.SetCurrentJustifiedCheckpoint(state.CurrentJustifiedCheckpoint()); err != nil {
		return nil, errors.Wrap(err, "could not set current justified checkpoint")
	}
	if err = newState.SetFinalizedCheckpoint(state.FinalizedCheckpoint()); err != nil {
		return nil, errors.Wrap(err, "could not set finalized checkpoint")
	}
	if err := newState.SetCurrentSyncCommittee(currentSyncCommittee); err != nil {
		return nil, errors.Wrap(err, "could not set current sync committee")
	}
	if err := newState.SetNextSyncCommittee(nextSyncCommittee); err != nil {
		return nil, errors.Wrap(err, "could not set next sync committee")
	}
	if err := newState.SetLatestExecutionPayloadHeader(&ethpb.ExecutionPayloadHeader{
		ParentHash:       make([]byte, 32),
		FeeRecipient:     make([]byte, 20),
		StateRoot:        make([]byte, 32),
		ReceiptRoot:      make([]byte, 32),
		LogsBloom:        make([]byte, 256),
		Random:           make([]byte, 32),
		BlockNumber:      0,
		GasLimit:         0,
		GasUsed:          0,
		Timestamp:        0,
		BaseFeePerGas:    make([]byte, 32),
		BlockHash:        make([]byte, 32),
		TransactionsRoot: make([]byte, 32),
	}); err != nil {
		return nil, errors.Wrap(err, "could not set latest execution payload header")
	}

	return newState, nil
}

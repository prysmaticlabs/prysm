package epbs

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
)

func processPayloadStateTransition(
	ctx context.Context,
	preState state.BeaconState,
	envelope interfaces.ROExecutionPayloadEnvelope,
) error {
	payload, err := envelope.Execution()
	if err != nil {
		return err
	}
	exe, ok := payload.(interfaces.ExecutionDataElectra)
	if !ok {
		return errors.New("could not cast execution data to electra execution data")
	}
	preState, err = electra.ProcessDepositRequests(ctx, preState, exe.DepositRequests())
	if err != nil {
		return errors.Wrap(err, "could not process deposit receipts")
	}
	preState, err = electra.ProcessWithdrawalRequests(ctx, preState, exe.WithdrawalRequests())
	if err != nil {
		return errors.Wrap(err, "could not process execution layer withdrawal requests")
	}
	if err := electra.ProcessConsolidationRequests(ctx, preState, exe.ConsolidationRequests()); err != nil {
		return errors.Wrap(err, "could not process consolidation requests")
	}
	if err := preState.SetLatestBlockHash(payload.BlockHash()); err != nil {
		return err
	}
	return preState.SetLatestFullSlot(preState.Slot())
}

func validateAgainstHeader(
	ctx context.Context,
	preState state.BeaconState,
	envelope interfaces.ROExecutionPayloadEnvelope,
) error {
	blockHeader := preState.LatestBlockHeader()
	if blockHeader == nil {
		return errors.New("invalid nil latest block header")
	}
	if len(blockHeader.StateRoot) == 0 || [32]byte(blockHeader.StateRoot) == [32]byte{} {
		prevStateRoot, err := preState.HashTreeRoot(ctx)
		if err != nil {
			return errors.Wrap(err, "could not compute previous state root")
		}
		blockHeader.StateRoot = prevStateRoot[:]
		if err := preState.SetLatestBlockHeader(blockHeader); err != nil {
			return errors.Wrap(err, "could not set latest block header")
		}
	}
	blockHeaderRoot, err := blockHeader.HashTreeRoot()
	if err != nil {
		return err
	}
	beaconBlockRoot, err := envelope.BeaconBlockRoot()
	if err != nil {
		return err
	}
	if blockHeaderRoot != beaconBlockRoot {
		return errors.New("beacon block root does not match previous header")
	}
	return nil
}

func validateAgainstCommittedBid(
	committedHeader *enginev1.ExecutionPayloadHeaderEPBS,
	envelope interfaces.ROExecutionPayloadEnvelope,
) error {
	builderIndex, err := envelope.BuilderIndex()
	if err != nil {
		return err
	}
	if committedHeader.BuilderIndex != builderIndex {
		return errors.New("builder index does not match committed header")
	}
	kzgRoot, err := envelope.BlobKzgCommitmentsRoot()
	if err != nil {
		return err
	}
	if [32]byte(committedHeader.BlobKzgCommitmentsRoot) != kzgRoot {
		return errors.New("blob KZG commitments root does not match committed header")
	}
	return nil
}

func checkPostStateRoot(
	ctx context.Context,
	preState state.BeaconState,
	envelope interfaces.ROExecutionPayloadEnvelope,
) error {
	stateRoot, err := preState.HashTreeRoot(ctx)
	if err != nil {
		return err
	}
	envelopeStateRoot, err := envelope.StateRoot()
	if err != nil {
		return err
	}
	if stateRoot != envelopeStateRoot {
		return errors.New("state root mismatch")
	}
	return nil
}

func ValidatePayloadStateTransition(
	ctx context.Context,
	preState state.BeaconState,
	envelope interfaces.ROExecutionPayloadEnvelope,
) error {
	if err := validateAgainstHeader(ctx, preState, envelope); err != nil {
		return err
	}
	committedHeader, err := preState.LatestExecutionPayloadHeaderEPBS()
	if err != nil {
		return err
	}
	if err := validateAgainstCommittedBid(committedHeader, envelope); err != nil {
		return err
	}
	if err := processPayloadStateTransition(ctx, preState, envelope); err != nil {
		return err
	}
	return checkPostStateRoot(ctx, preState, envelope)
}

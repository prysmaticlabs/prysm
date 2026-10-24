package eip8453

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/altair"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/blocks"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/gloas"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/validators"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/pkg/errors"
)

// ProcessBlock applies a block, leaving the randao and BLS change signatures to the caller's batch.
func ProcessBlock(ctx context.Context, st state.BeaconState, blk interfaces.ReadOnlyBeaconBlock) (state.BeaconState, error) {
	if err := gloas.ProcessParentExecutionPayload(ctx, st, blk); err != nil {
		return nil, errors.Wrap(err, "could not process parent execution payload")
	}
	bodyRoot, err := blk.Body().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not hash tree root beacon block body")
	}
	parentRoot := blk.ParentRoot()
	st, err = blocks.ProcessBlockHeaderNoVerify(ctx, st, blk.Slot(), blk.ProposerIndex(), parentRoot[:], bodyRoot[:])
	if err != nil {
		return nil, errors.Wrap(err, "could not process block header")
	}
	if err := gloas.ProcessWithdrawals(st); err != nil {
		return nil, errors.Wrap(err, "could not process withdrawals")
	}
	if err := gloas.ProcessExecutionPayloadBid(st, blk); err != nil {
		return nil, errors.Wrap(err, "could not process execution payload bid")
	}
	reveal := blk.Body().RandaoReveal()
	st, err = blocks.ProcessRandaoNoVerify(st, reveal[:])
	if err != nil {
		return nil, errors.Wrap(err, "could not process randao")
	}
	if err := ProcessOperations(ctx, st, blk); err != nil {
		return nil, errors.Wrap(err, "could not process operations")
	}
	sa, err := blk.Body().SyncAggregate()
	if err != nil {
		return nil, err
	}
	st, _, err = altair.ProcessSyncAggregate(ctx, st, sa)
	if err != nil {
		return nil, errors.Wrap(err, "could not process sync aggregate")
	}
	if err := ProcessHeightEvents(ctx, st); err != nil {
		return nil, errors.Wrap(err, "could not process height events")
	}
	return st, nil
}

// ProcessOperations checks the operation limits and support bits, then applies the operations in spec order.
func ProcessOperations(ctx context.Context, st state.BeaconState, blk interfaces.ReadOnlyBeaconBlock) error {
	body := blk.Body()
	attesterSlashings, err := body.AttesterSlashingsEIP8453()
	if err != nil {
		return err
	}
	legacySlashings, err := body.LegacyAttesterSlashings()
	if err != nil {
		return err
	}
	attestations, err := body.AttestationsEIP8453()
	if err != nil {
		return err
	}
	changes, err := body.BLSToExecutionChanges()
	if err != nil {
		return err
	}
	payloadAttestations, err := body.PayloadAttestations()
	if err != nil {
		return err
	}
	headVotes, err := body.HeadVotes()
	if err != nil {
		return err
	}
	support, err := body.HeadVoteSupportBits()
	if err != nil {
		return err
	}
	cfg := params.BeaconConfig()
	limits := []struct {
		name       string
		count, max uint64
	}{
		{"proposer slashings", uint64(len(body.ProposerSlashings())), cfg.MaxProposerSlashings},
		{"attester slashings", uint64(len(attesterSlashings)), cfg.MaxAttesterSlashingsElectra},
		{"legacy attester slashings", uint64(len(legacySlashings)), cfg.MaxAttesterSlashingsElectra},
		{"attestations", uint64(len(attestations)), cfg.MaxAttestationsElectra},
		{"voluntary exits", uint64(len(body.VoluntaryExits())), cfg.MaxVoluntaryExits},
		{"bls to execution changes", uint64(len(changes)), cfg.MaxBlsToExecutionChanges},
		{"payload attestations", uint64(len(payloadAttestations)), fieldparams.MaxPayloadAttestations},
		{"head votes", uint64(len(headVotes)), cfg.MaxHeadVotes},
	}
	for _, limit := range limits {
		if limit.count > limit.max {
			return fmt.Errorf("%d %s, over the limit of %d", limit.count, limit.name, limit.max)
		}
	}
	if support.Len() != uint64(len(headVotes)) {
		return fmt.Errorf("%d head vote support bits for %d head votes", support.Len(), len(headVotes))
	}
	for i, vote := range headVotes {
		if support.BitAt(uint64(i)) && (st.Slot() == 0 || vote.Data.Slot != st.Slot()-1) {
			return fmt.Errorf("supported head vote %d is for slot %d, not the previous slot", i, vote.Data.Slot)
		}
	}

	if len(body.ProposerSlashings()) > 0 {
		exitInfo := validators.ExitInformation(st)
		for _, slashing := range body.ProposerSlashings() {
			if err := blocks.VerifyProposerSlashing(st, slashing); err != nil {
				return errors.Wrap(err, "could not verify proposer slashing")
			}
			if err := slashValidator(ctx, st, slashing.Header_1.Header.ProposerIndex, exitInfo); err != nil {
				return err
			}
		}
	}
	for _, slashing := range attesterSlashings {
		if err := ProcessAttesterSlashing(ctx, st, slashing); err != nil {
			return err
		}
	}
	for _, slashing := range legacySlashings {
		if err := ProcessLegacyAttesterSlashing(ctx, st, slashing); err != nil {
			return err
		}
	}
	for _, att := range attestations {
		if err := ProcessAttestation(ctx, st, att); err != nil {
			return err
		}
	}
	if len(body.VoluntaryExits()) > 0 {
		if _, err := blocks.ProcessVoluntaryExits(ctx, st, body.VoluntaryExits(), validators.ExitInformation(st)); err != nil {
			return err
		}
	}
	if _, err := blocks.ProcessBLSToExecutionChanges(st, blk); err != nil {
		return err
	}
	if err := gloas.ProcessPayloadAttestations(ctx, st, body); err != nil {
		return err
	}
	for _, vote := range headVotes {
		if err := ProcessHeadVote(ctx, st, vote); err != nil {
			return err
		}
	}
	return nil
}

// FillTargetRoot sets the current target's root to the previous block root, the block that advanced the height.
func FillTargetRoot(st state.BeaconState, blockRoot [32]byte) error {
	target, err := st.TargetCheckpoint()
	if err != nil {
		return err
	}
	if !isZeroRoot(target.Root) {
		return nil
	}
	return st.SetTargetCheckpoint(&ethpb.HeightCheckpoint{Height: target.Height, Root: blockRoot[:]})
}

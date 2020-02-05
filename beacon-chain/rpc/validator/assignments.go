package validator

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	st "github.com/prysmaticlabs/prysm/beacon-chain/state"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetDuties returns the committee assignment response from a given validator public key.
// The committee assignment response contains the following fields for the current and previous epoch:
//	1.) The list of validators in the committee.
//	2.) The shard to which the committee is assigned.
//	3.) The slot at which the committee is assigned.
//	4.) The bool signaling if the validator is expected to propose a block at the assigned slot.
//  5.) Update the slashing pool from slashing server.
func (vs *Server) GetDuties(ctx context.Context, req *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Advance state with empty transitions up to the requested epoch start slot.
	if epochStartSlot := helpers.StartSlot(req.Epoch); s.Slot() < epochStartSlot {
		s, err = state.ProcessSlots(ctx, s, epochStartSlot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", epochStartSlot, err)
		}
	}

	committeeAssignments, proposerIndexToSlot, err := helpers.CommitteeAssignments(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}

	var validatorAssignments []*ethpb.DutiesResponse_Duty
	for _, pubKey := range req.PublicKeys {
		if ctx.Err() != nil {
			return nil, status.Errorf(codes.Aborted, "Could not continue fetching assignments: %v", ctx.Err())
		}
		// Default assignment.
		assignment := &ethpb.DutiesResponse_Duty{
			PublicKey: pubKey,
		}

		idx, ok, err := vs.BeaconDB.ValidatorIndex(ctx, pubKey)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch validator idx for public key %#x: %v", pubKey, err)
		}
		if ok {
			ca, ok := committeeAssignments[idx]
			if ok {
				assignment.Committee = ca.Committee
				assignment.Status = ethpb.ValidatorStatus_ACTIVE
				assignment.ValidatorIndex = idx
				assignment.PublicKey = pubKey
				assignment.AttesterSlot = ca.AttesterSlot
				assignment.ProposerSlot = proposerIndexToSlot[idx]
				assignment.CommitteeIndex = ca.CommitteeIndex
			}
		}

		validatorAssignments = append(validatorAssignments, assignment)
	}
	if vs.SlasherClient != nil {
		go vs.updateSlashingPool(ctx, s)
	}
	return &ethpb.DutiesResponse{
		Duties: validatorAssignments,
	}, nil
}

func (vs *Server) updateSlashingPool(ctx context.Context, state *st.BeaconState) error {
	psr, err := vs.SlasherClient.ProposerSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active})
	if err != nil {
		return status.Errorf(codes.Internal, "Could not retrieve proposer slashings: %v", err)
	}
	asr, err := vs.SlasherClient.AttesterSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active})
	if err != nil {
		return status.Errorf(codes.Internal, "Could not retrieve attester slashings: %v", err)
	}
	for _, ps := range psr.ProposerSlashing {
		vs.SlashingPool.InsertProposerSlashing(ctx, state, ps)
	}
	for _, as := range asr.AttesterSlashing {
		vs.SlashingPool.InsertAttesterSlashing(ctx, state, as)
	}
	return nil
}

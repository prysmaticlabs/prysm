package validator

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetDuties returns the committee assignment response from a given validator public key.
// The committee assignment response contains the following fields for the current and previous epoch:
//	1.) The list of validators in the committee.
//	2.) The shard to which the committee is assigned.
//	3.) The slot at which the committee is assigned.
//	4.) The bool signaling if the validator is expected to propose a block at the assigned slot.
func (vs *Server) GetDuties(ctx context.Context, req *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Advance state with empty transitions up to the requested epoch start slot.
	if epochStartSlot := helpers.StartSlot(req.Epoch); s.Slot < epochStartSlot {
		s, err = state.ProcessSlots(ctx, s, epochStartSlot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", epochStartSlot, err)
		}
	}

	var assignments []*ethpb.DutiesResponse_Duty
	for _, pubKey := range req.PublicKeys {
		if ctx.Err() != nil {
			return nil, status.Errorf(codes.Aborted, "Could not continue fetching assignments: %v", ctx.Err())
		}
		// Default assignment.
		assignment := &ethpb.DutiesResponse_Duty{
			PublicKey: pubKey,
		}

		idx, ok, err := vs.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch validator idx for public key %#x: %v", pubKey, err)
		}
		if ok {
			st := vs.assignmentStatus(idx, s)
			assignment.Status = st
			if st == ethpb.ValidatorStatus_ACTIVE {
				assignment, err = vs.assignment(idx, s, req.Epoch)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "Could not fetch assignment for public key %#x: %v", pubKey, err)
				}
				assignment.PublicKey = pubKey
			}
		}
		assignments = append(assignments, assignment)
	}

	return &ethpb.DutiesResponse{
		Duties: assignments,
	}, nil
}

func (vs *Server) assignment(idx uint64, beaconState *pbp2p.BeaconState, epoch uint64) (*ethpb.DutiesResponse_Duty, error) {
	committee, committeeIndex, aSlot, pSlot, err := helpers.CommitteeAssignment(beaconState, epoch, idx)
	if err != nil {
		return nil, err
	}
	return &ethpb.DutiesResponse_Duty{
		Committee:      committee,
		CommitteeIndex: committeeIndex,
		AttesterSlot:   aSlot,
		ProposerSlot:   pSlot,
		Status:         vs.assignmentStatus(idx, beaconState),
	}, nil
}

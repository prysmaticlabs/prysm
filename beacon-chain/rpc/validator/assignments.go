package validator

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CommitteeAssignment returns the committee assignment response from a given validator public key.
// The committee assignment response contains the following fields for the current and previous epoch:
//	1.) The list of validators in the committee.
//	2.) The shard to which the committee is assigned.
//	3.) The slot at which the committee is assigned.
//	4.) The bool signaling if the validator is expected to propose a block at the assigned slot.
func (vs *Server) CommitteeAssignment(ctx context.Context, req *pb.AssignmentRequest) (*pb.AssignmentResponse, error) {
	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Advance state with empty transitions up to the requested epoch start slot.
	if epochStartSlot := helpers.StartSlot(req.EpochStart); s.Slot < epochStartSlot {
		s, err = state.ProcessSlots(ctx, s, epochStartSlot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", epochStartSlot, err)
		}
	}

	var assignments []*pb.AssignmentResponse_ValidatorAssignment
	for _, pubKey := range req.PublicKeys {
		if ctx.Err() != nil {
			return nil, status.Errorf(codes.Aborted, "Could not continue fetching assignments: %v", ctx.Err())
		}
		// Default assignment.
		assignment := &pb.AssignmentResponse_ValidatorAssignment{
			PublicKey: pubKey,
			Status:    pb.ValidatorStatus_UNKNOWN_STATUS,
		}

		idx, ok, err := vs.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch validator idx for public key %#x: %v", pubKey, err)
		}
		if ok {
			st := vs.assignmentStatus(idx, s)
			assignment.Status = st
			if st == pb.ValidatorStatus_ACTIVE {
				assignment, err = vs.assignment(idx, s, req.EpochStart)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "Could not fetch assignment for public key %#x: %v", pubKey, err)
				}
				assignment.PublicKey = pubKey
			}
		}
		assignments = append(assignments, assignment)
	}

	return &pb.AssignmentResponse{
		ValidatorAssignment: assignments,
	}, nil
}

func (vs *Server) assignment(idx uint64, beaconState *pbp2p.BeaconState, epoch uint64) (*pb.AssignmentResponse_ValidatorAssignment, error) {
	committee, committeeIndex, aSlot, pSlot, err := helpers.CommitteeAssignment(beaconState, epoch, idx)
	if err != nil {
		return nil, err
	}
	return &pb.AssignmentResponse_ValidatorAssignment{
		Committee:      committee,
		CommitteeIndex: committeeIndex,
		AttesterSlot:   aSlot,
		ProposerSlot:   pSlot,
		Status:         vs.assignmentStatus(idx, beaconState),
	}, nil
}

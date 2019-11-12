package validator

import (
	"context"

	"github.com/pkg/errors"
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
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	var err error
	s := vs.HeadFetcher.HeadState()
	// Advance state with empty transitions up to the requested epoch start slot.
	if epochStartSlot := helpers.StartSlot(req.EpochStart); s.Slot < epochStartSlot {
		s, err = state.ProcessSlots(ctx, s, epochStartSlot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process slots up to %d", epochStartSlot)
		}
	}

	var assignments []*pb.AssignmentResponse_ValidatorAssignment
	for _, pubKey := range req.PublicKeys {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Default assignment
		assignment := &pb.AssignmentResponse_ValidatorAssignment{
			PublicKey: pubKey,
			Status:    pb.ValidatorStatus_UNKNOWN_STATUS,
		}

		idx, ok, err := vs.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, err
		}
		if ok {
			status := vs.assignmentStatus(uint64(idx), s)
			assignment.Status = status
			if status == pb.ValidatorStatus_ACTIVE {
				assignment, err = vs.assignment(uint64(idx), s, req.EpochStart)
				if err != nil {
					return nil, err
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
	status := vs.assignmentStatus(idx, beaconState)
	return &pb.AssignmentResponse_ValidatorAssignment{
		Committee:      committee,
		CommitteeIndex: committeeIndex,
		AttesterSlot:   aSlot,
		ProposerSlot:   pSlot,
		Status:         status,
	}, nil
}

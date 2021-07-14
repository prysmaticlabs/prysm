package validator

import (
	"context"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	statev1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetAttesterDuties requests the beacon node to provide a set of attestation duties,
// which should be performed by validators, for a particular epoch.
func (vs *Server) GetAttesterDuties(ctx context.Context, req *v1.AttesterDutiesRequest) (*v1.AttesterDutiesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validatorv1.GetAttesterDuties")
	defer span.End()

	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	currentEpoch := helpers.SlotToEpoch(vs.TimeFetcher.CurrentSlot())
	if req.Epoch > currentEpoch+1 {
		return nil, status.Errorf(codes.InvalidArgument, "Request epoch %d can not be greater than next epoch %d", req.Epoch, currentEpoch+1)
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	vals := make([]iface.ReadOnlyValidator, len(req.Index))
	for i, index := range req.Index {
		val, err := s.ValidatorAtIndexReadOnly(index)
		if err != nil {
			if _, ok := err.(*statev1.ValidatorIndexOutOfRangeError); ok {
				return nil, status.Errorf(codes.InvalidArgument, "Invalid index: %v", err)
			} else {
				return nil, status.Errorf(codes.Internal, "Could not get validator: %v", err)
			}
		}
		vals[i] = val
	}

	// Advance state with empty transitions up to the requested epoch start slot.
	epochStartSlot, err := helpers.StartSlot(req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not obtain epoch's start slot: %v", err)
	}
	if s.Slot() < epochStartSlot {
		s, err = state.ProcessSlots(ctx, s, epochStartSlot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", epochStartSlot, err)
		}
	}

	committeeAssignments, _, err := helpers.CommitteeAssignments(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}
	var distinctCommitteeIndexes []types.CommitteeIndex
assignmentLoop:
	for _, assignment := range committeeAssignments {
		for _, index := range distinctCommitteeIndexes {
			if index == assignment.CommitteeIndex {
				continue assignmentLoop
			}
		}
		distinctCommitteeIndexes = append(distinctCommitteeIndexes, assignment.CommitteeIndex)
	}

	duties := make([]*v1.AttesterDuty, len(req.Index))
	for i, index := range req.Index {
		pubkey := vals[i].PublicKey()
		committee := committeeAssignments[index]
		var valIndexInCommittee types.CommitteeIndex
		for cIndex, vIndex := range committee.Committee {
			if vIndex == index {
				valIndexInCommittee = types.CommitteeIndex(uint64(cIndex))
				break
			}
		}
		duties[i] = &v1.AttesterDuty{
			Pubkey:                  pubkey[:],
			ValidatorIndex:          index,
			CommitteeIndex:          committee.CommitteeIndex,
			CommitteeLength:         uint64(len(committee.Committee)),
			CommitteesAtSlot:        uint64(len(distinctCommitteeIndexes)),
			ValidatorCommitteeIndex: valIndexInCommittee,
			Slot:                    epochStartSlot,
		}
	}

	dependentRoot, err := helpers.BlockRootAtSlot(s, epochStartSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block root at slot %d: %v", epochStartSlot, err)
	}

	return &v1.AttesterDutiesResponse{
		DependentRoot: dependentRoot,
		Data:          duties,
	}, nil
}

// GetProposerDuties requests beacon node to provide all validators that are scheduled to propose a block in the given epoch.
func (vs *Server) GetProposerDuties(ctx context.Context, req *v1.ProposerDutiesRequest) (*v1.ProposerDutiesResponse, error) {
	return nil, errors.New("Unimplemented")
}

// ProduceBlock requests the beacon node to produce a valid unsigned beacon block, which can then be signed by a proposer and submitted.
func (vs *Server) ProduceBlock(ctx context.Context, req *v1.ProduceBlockRequest) (*v1.ProduceBlockResponse, error) {
	return nil, errors.New("Unimplemented")
}

// ProduceAttestationData requests that the beacon node produces attestation data for
// the requested committee index and slot based on the nodes current head.
func (vs *Server) ProduceAttestationData(ctx context.Context, req *v1.ProduceAttestationDataRequest) (*v1.ProduceAttestationDataResponse, error) {
	return nil, errors.New("Unimplemented")
}

// GetAggregateAttestation aggregates all attestations matching the given attestation data root and slot, returning the aggregated result.
func (vs *Server) GetAggregateAttestation(ctx context.Context, req *v1.AggregateAttestationRequest) (*v1.AggregateAttestationResponse, error) {
	return nil, errors.New("Unimplemented")
}

// SubmitAggregateAndProofs verifies given aggregate and proofs and publishes them on appropriate gossipsub topic.
func (vs *Server) SubmitAggregateAndProofs(ctx context.Context, req *v1.SubmitAggregateAndProofsRequest) (*emptypb.Empty, error) {
	return nil, errors.New("Unimplemented")
}

// SubmitBeaconCommitteeSubscription searches using discv5 for peers related to the provided subnet information
// and replaces current peers with those ones if necessary.
func (vs *Server) SubmitBeaconCommitteeSubscription(ctx context.Context, req *v1.SubmitBeaconCommitteeSubscriptionsRequest) (*emptypb.Empty, error) {
	return nil, errors.New("Unimplemented")
}

package validator

import (
	"bytes"
	"context"
	"sort"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	statev1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/proto/migration"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
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

	cs := vs.TimeFetcher.CurrentSlot()
	currentEpoch := helpers.SlotToEpoch(cs)
	if req.Epoch > currentEpoch+1 {
		return nil, status.Errorf(codes.InvalidArgument, "Request epoch %d can not be greater than next epoch %d", req.Epoch, currentEpoch+1)
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	s, err = advanceState(ctx, s, req.Epoch, currentEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not advance state to requested epoch start slot: %v", err)
	}

	committeeAssignments, _, err := helpers.CommitteeAssignments(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}
	activeValidatorCount, err := helpers.ActiveValidatorCount(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
	}
	committeesAtSlot := helpers.SlotCommitteeCount(activeValidatorCount)

	duties := make([]*v1.AttesterDuty, len(req.Index))
	for i, index := range req.Index {
		val, err := s.ValidatorAtIndexReadOnly(index)
		if _, ok := err.(*statev1.ValidatorIndexOutOfRangeError); ok {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid index: %v", err)
		} else if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator: %v", err)
		}
		pubkey := val.PublicKey()
		committee := committeeAssignments[index]
		var valIndexInCommittee types.CommitteeIndex
		// valIndexInCommittee will be 0 in case we don't get a match. This is a potential false positive,
		// however it's an impossible condition because every validator must be assigned to a committee.
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
			CommitteesAtSlot:        committeesAtSlot,
			ValidatorCommitteeIndex: valIndexInCommittee,
			Slot:                    committee.AttesterSlot,
		}
	}

	root, err := attestationDependentRoot(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get dependent root: %v", err)
	}

	return &v1.AttesterDutiesResponse{
		DependentRoot: root,
		Data:          duties,
	}, nil
}

// GetProposerDuties requests beacon node to provide all validators that are scheduled to propose a block in the given epoch.
func (vs *Server) GetProposerDuties(ctx context.Context, req *v1.ProposerDutiesRequest) (*v1.ProposerDutiesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validatorv1.GetProposerDuties")
	defer span.End()

	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	cs := vs.TimeFetcher.CurrentSlot()
	currentEpoch := helpers.SlotToEpoch(cs)
	if req.Epoch > currentEpoch {
		return nil, status.Errorf(codes.InvalidArgument, "Request epoch %d can not be greater than current epoch %d", req.Epoch, currentEpoch)
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	s, err = advanceState(ctx, s, req.Epoch, currentEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not advance state to requested epoch start slot: %v", err)
	}

	_, proposals, err := helpers.CommitteeAssignments(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}

	duties := make([]*v1.ProposerDuty, 0)
	for index, slots := range proposals {
		val, err := s.ValidatorAtIndexReadOnly(index)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator: %v", err)
		}
		pubkey48 := val.PublicKey()
		pubkey := pubkey48[:]
		for _, s := range slots {
			duties = append(duties, &v1.ProposerDuty{
				Pubkey:         pubkey,
				ValidatorIndex: index,
				Slot:           s,
			})
		}
	}
	sort.Slice(duties, func(i, j int) bool {
		return duties[i].Slot < duties[j].Slot
	})

	root, err := proposalDependentRoot(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get dependent root: %v", err)
	}

	return &v1.ProposerDutiesResponse{
		DependentRoot: root,
		Data:          duties,
	}, nil
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
	ctx, span := trace.StartSpan(ctx, "validatorv1.GetAggregateAttestation")
	defer span.End()

	allAtts := vs.AttestationsPool.AggregatedAttestations()
	var bestMatchingAtt *v1alpha1.Attestation
	for _, att := range allAtts {
		if att.Data.Slot == req.Slot {
			root, err := att.Data.HashTreeRoot()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get attestation data root: %v", err)
			}
			if bytes.Equal(root[:], req.AttestationDataRoot) {
				if bestMatchingAtt == nil || len(att.AggregationBits) > len(bestMatchingAtt.AggregationBits) {
					bestMatchingAtt = att
				}
			}
		}
	}

	if bestMatchingAtt == nil {
		return nil, status.Error(codes.InvalidArgument, "No matching attestation found")
	}
	return &v1.AggregateAttestationResponse{
		Data: migration.V1Alpha1AttestationToV1(bestMatchingAtt),
	}, nil
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

// attestationDependentRoot is get_block_root_at_slot(state, compute_start_slot_at_epoch(epoch - 1) - 1)
// or the genesis block root in the case of underflow.
func attestationDependentRoot(s iface.BeaconState, epoch types.Epoch) ([]byte, error) {
	var dependentRootSlot types.Slot
	if epoch <= 1 {
		dependentRootSlot = 0
	} else {
		prevEpochStartSlot, err := helpers.StartSlot(epoch.Sub(1))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not obtain epoch's start slot: %v", err)
		}
		dependentRootSlot = prevEpochStartSlot.Sub(1)
	}
	root, err := helpers.BlockRootAtSlot(s, dependentRootSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get block root")
	}
	return root, nil
}

// proposalDependentRoot is get_block_root_at_slot(state, compute_start_slot_at_epoch(epoch) - 1)
// or the genesis block root in the case of underflow.
func proposalDependentRoot(s iface.BeaconState, epoch types.Epoch) ([]byte, error) {
	var dependentRootSlot types.Slot
	if epoch == 0 {
		dependentRootSlot = 0
	} else {
		epochStartSlot, err := helpers.StartSlot(epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not obtain epoch's start slot: %v", err)
		}
		dependentRootSlot = epochStartSlot.Sub(1)
	}
	root, err := helpers.BlockRootAtSlot(s, dependentRootSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get block root")
	}
	return root, nil
}

// advanceState advances state with empty transitions up to the requested epoch start slot.
// In case 1 epoch ahead was requested, we take the start slot of the current epoch.
// Taking the start slot of the next epoch would result in an error inside state.ProcessSlots.
func advanceState(ctx context.Context, s iface.BeaconState, requestedEpoch, currentEpoch types.Epoch) (iface.BeaconState, error) {
	var epochStartSlot types.Slot
	var err error
	if requestedEpoch == currentEpoch+1 {
		epochStartSlot, err = helpers.StartSlot(requestedEpoch.Sub(1))
		if err != nil {
			return nil, errors.Wrap(err, "Could not obtain epoch's start slot")
		}
	} else {
		epochStartSlot, err = helpers.StartSlot(requestedEpoch)
		if err != nil {
			return nil, errors.Wrap(err, "Could not obtain epoch's start slot")
		}
	}
	if s.Slot() < epochStartSlot {
		s, err = state.ProcessSlots(ctx, s, epochStartSlot)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not process slots up to %d", epochStartSlot)
		}
	}

	return s, nil
}

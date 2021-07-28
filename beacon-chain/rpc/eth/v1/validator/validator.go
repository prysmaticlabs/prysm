package validator

import (
	"bytes"
	"context"
	"sort"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	core "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	statev1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/proto/migration"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
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
	ctx, span := trace.StartSpan(ctx, "validatorv1.ProduceBlock")
	defer span.End()

	v1alpha1req := &v1alpha1.BlockRequest{
		Slot:         req.Slot,
		RandaoReveal: req.RandaoReveal,
		Graffiti:     req.Graffiti,
	}
	v1alpha1resp, err := vs.V1Alpha1Server.GetBlock(ctx, v1alpha1req)
	if err != nil {
		// We simply return err because it's already of a gRPC error type.
		return nil, err
	}
	block, err := migration.V1Alpha1ToV1Block(v1alpha1resp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
	}

	return &v1.ProduceBlockResponse{Data: block}, nil
}

// ProduceAttestationData requests that the beacon node produces attestation data for
// the requested committee index and slot based on the nodes current head.
func (vs *Server) ProduceAttestationData(ctx context.Context, req *v1.ProduceAttestationDataRequest) (*v1.ProduceAttestationDataResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validatorv1.ProduceAttestationData")
	defer span.End()

	v1alpha1req := &v1alpha1.AttestationDataRequest{
		Slot:           req.Slot,
		CommitteeIndex: req.CommitteeIndex,
	}
	v1alpha1resp, err := vs.V1Alpha1Server.GetAttestationData(ctx, v1alpha1req)
	if err != nil {
		// We simply return err because it's already of a gRPC error type.
		return nil, err
	}
	attData := migration.V1Alpha1AttDataToV1(v1alpha1resp)

	return &v1.ProduceAttestationDataResponse{Data: attData}, nil
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
	ctx, span := trace.StartSpan(ctx, "validatorv1.GetAggregateAttestation")
	defer span.End()

	for _, agg := range req.Data {
		if agg == nil || agg.Message == nil || agg.Message.Aggregate == nil || agg.Message.Aggregate.Data == nil {
			return nil, status.Error(codes.InvalidArgument, "Signed aggregate request can't be nil")
		}
		sigLen := params.BeaconConfig().BLSSignatureLength
		emptySig := make([]byte, sigLen)
		if bytes.Equal(agg.Signature, emptySig) || bytes.Equal(agg.Message.SelectionProof, emptySig) {
			return nil, status.Error(codes.InvalidArgument, "Signed signatures can't be zero hashes")
		}
		if len(agg.Signature) != sigLen || len(agg.Message.Aggregate.Signature) != sigLen {
			return nil, status.Errorf(codes.InvalidArgument, "Incorrect signature length. Expected %d bytes", sigLen)
		}

		// As a preventive measure, a beacon node shouldn't broadcast an attestation whose slot is out of range.
		if err := helpers.ValidateAttestationTime(agg.Message.Aggregate.Data.Slot,
			vs.TimeFetcher.GenesisTime(), params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
			return nil, status.Error(codes.InvalidArgument, "Attestation slot is no longer valid from current time")
		}
	}

	broadcastFailed := false
	for _, agg := range req.Data {
		v1alpha1Agg := migration.V1SignedAggregateAttAndProofToV1Alpha1(agg)
		if err := vs.Broadcaster.Broadcast(ctx, v1alpha1Agg); err != nil {
			broadcastFailed = true
		} else {
			log.WithFields(logrus.Fields{
				"slot":            agg.Message.Aggregate.Data.Slot,
				"committeeIndex":  agg.Message.Aggregate.Data.Index,
				"validatorIndex":  agg.Message.AggregatorIndex,
				"aggregatedCount": agg.Message.Aggregate.AggregationBits.Count(),
			}).Debug("Broadcasting aggregated attestation and proof")
		}
	}

	if broadcastFailed {
		return nil, status.Errorf(
			codes.Internal,
			"Could not broadcast one or more signed aggregated attestations.")
	}

	return &emptypb.Empty{}, nil
}

// SubmitBeaconCommitteeSubscription searches using discv5 for peers related to the provided subnet information
// and replaces current peers with those ones if necessary.
func (vs *Server) SubmitBeaconCommitteeSubscription(ctx context.Context, req *v1.SubmitBeaconCommitteeSubscriptionsRequest) (*emptypb.Empty, error) {
	return nil, errors.New("Unimplemented")
}

// attestationDependentRoot is get_block_root_at_slot(state, compute_start_slot_at_epoch(epoch - 1) - 1)
// or the genesis block root in the case of underflow.
func attestationDependentRoot(s state.BeaconState, epoch types.Epoch) ([]byte, error) {
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
func proposalDependentRoot(s state.BeaconState, epoch types.Epoch) ([]byte, error) {
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
func advanceState(ctx context.Context, s state.BeaconState, requestedEpoch, currentEpoch types.Epoch) (state.BeaconState, error) {
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
		s, err = core.ProcessSlots(ctx, s, epochStartSlot)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not process slots up to %d", epochStartSlot)
		}
	}

	return s, nil
}

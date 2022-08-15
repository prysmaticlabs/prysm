package validator

import (
	"bytes"
	"context"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubmitAggregateSelectionProof is called by a validator when its assigned to be an aggregator.
// The aggregator submits the selection proof to obtain the aggregated attestation
// object to sign over.
func (vs *Server) SubmitAggregateSelectionProof(ctx context.Context, req *ethpb.AggregateSelectionRequest) (*ethpb.AggregateSelectionResponse, error) {
	ctx, span := trace.StartSpan(ctx, "AggregatorServer.SubmitAggregateSelectionProof")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	if vs.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	// An optimistic validator MUST NOT participate in attestation. (i.e., sign across the DOMAIN_BEACON_ATTESTER, DOMAIN_SELECTION_PROOF or DOMAIN_AGGREGATE_AND_PROOF domains).
	if err := vs.optimisticStatus(ctx); err != nil {
		return nil, err
	}

	st, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine head state: %v", err)
	}

	validatorIndex, exists := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(req.PublicKey))
	if !exists {
		return nil, status.Error(codes.Internal, "Could not locate validator index in DB")
	}

	epoch := slots.ToEpoch(req.Slot)
	activeValidatorIndices, err := helpers.ActiveValidatorIndices(ctx, st, epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get validators: %v", err)
	}
	seed, err := helpers.Seed(st, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get seed: %v", err)
	}
	committee, err := helpers.BeaconCommittee(ctx, activeValidatorIndices, seed, req.Slot, req.CommitteeIndex)
	if err != nil {
		return nil, err
	}

	// Check if the validator is an aggregator
	isAggregator, err := helpers.IsAggregator(uint64(len(committee)), req.SlotSignature)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get aggregator status: %v", err)
	}
	if !isAggregator {
		return nil, status.Errorf(codes.InvalidArgument, "Validator is not an aggregator")
	}

	if err := vs.AttPool.AggregateUnaggregatedAttestationsBySlotIndex(ctx, req.Slot, req.CommitteeIndex); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not aggregate unaggregated attestations")
	}
	aggregatedAtts := vs.AttPool.AggregatedAttestationsBySlotIndex(ctx, req.Slot, req.CommitteeIndex)

	// Filter out the best aggregated attestation (ie. the one with the most aggregated bits).
	if len(aggregatedAtts) == 0 {
		aggregatedAtts = vs.AttPool.UnaggregatedAttestationsBySlotIndex(ctx, req.Slot, req.CommitteeIndex)
		if len(aggregatedAtts) == 0 {
			return nil, status.Errorf(codes.NotFound, "Could not find attestation for slot and committee in pool")
		}
	}

	var indexInCommittee uint64
	for i, idx := range committee {
		if idx == validatorIndex {
			indexInCommittee = uint64(i)
		}
	}

	best := aggregatedAtts[0]
	for _, aggregatedAtt := range aggregatedAtts[1:] {
		// The aggregator should prefer an attestation that they have signed. We check this by
		// looking at the attestation's committee index against the validator's committee index
		// and check the aggregate bits to ensure the validator's index is set.
		if aggregatedAtt.Data.CommitteeIndex == req.CommitteeIndex &&
			aggregatedAtt.AggregationBits.BitAt(indexInCommittee) &&
			(!best.AggregationBits.BitAt(indexInCommittee) ||
				aggregatedAtt.AggregationBits.Count() > best.AggregationBits.Count()) {
			best = aggregatedAtt
		}

		// If the "best" still doesn't contain the validator's index, check the aggregation bits to
		// choose the attestation with the most bits set.
		if !best.AggregationBits.BitAt(indexInCommittee) &&
			aggregatedAtt.AggregationBits.Count() > best.AggregationBits.Count() {
			best = aggregatedAtt
		}
	}
	a := &ethpb.AggregateAttestationAndProof{
		Aggregate:       best,
		SelectionProof:  req.SlotSignature,
		AggregatorIndex: validatorIndex,
	}
	return &ethpb.AggregateSelectionResponse{AggregateAndProof: a}, nil
}

// SubmitSignedAggregateSelectionProof is called by a validator to broadcast a signed
// aggregated and proof object.
func (vs *Server) SubmitSignedAggregateSelectionProof(
	ctx context.Context,
	req *ethpb.SignedAggregateSubmitRequest,
) (*ethpb.SignedAggregateSubmitResponse, error) {
	if req.SignedAggregateAndProof == nil || req.SignedAggregateAndProof.Message == nil ||
		req.SignedAggregateAndProof.Message.Aggregate == nil || req.SignedAggregateAndProof.Message.Aggregate.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "Signed aggregate request can't be nil")
	}
	emptySig := make([]byte, fieldparams.BLSSignatureLength)
	if bytes.Equal(req.SignedAggregateAndProof.Signature, emptySig) ||
		bytes.Equal(req.SignedAggregateAndProof.Message.SelectionProof, emptySig) {
		return nil, status.Error(codes.InvalidArgument, "Signed signatures can't be zero hashes")
	}

	// As a preventive measure, a beacon node shouldn't broadcast an attestation whose slot is out of range.
	if err := helpers.ValidateAttestationTime(req.SignedAggregateAndProof.Message.Aggregate.Data.Slot,
		vs.TimeFetcher.GenesisTime(), params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Attestation slot is no longer valid from current time")
	}

	if err := vs.P2P.Broadcast(ctx, req.SignedAggregateAndProof); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast signed aggregated attestation: %v", err)
	}

	log.WithFields(logrus.Fields{
		"slot":            req.SignedAggregateAndProof.Message.Aggregate.Data.Slot,
		"committeeIndex":  req.SignedAggregateAndProof.Message.Aggregate.Data.CommitteeIndex,
		"validatorIndex":  req.SignedAggregateAndProof.Message.AggregatorIndex,
		"aggregatedCount": req.SignedAggregateAndProof.Message.Aggregate.AggregationBits.Count(),
	}).Debug("Broadcasting aggregated attestation and proof")

	return &ethpb.SignedAggregateSubmitResponse{}, nil
}

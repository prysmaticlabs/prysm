package validator

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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

	st, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine head state: %v", err)
	}

	validatorIndex, exists := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(req.PublicKey))
	if !exists {
		return nil, status.Error(codes.Internal, "Could not locate validator index in DB")
	}

	epoch := helpers.SlotToEpoch(req.Slot)
	activeValidatorIndices, err := helpers.ActiveValidatorIndices(st, epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get validators: %v", err)
	}
	seed, err := helpers.Seed(st, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get seed: %v", err)
	}
	committee, err := helpers.BeaconCommittee(activeValidatorIndices, seed, req.Slot, req.CommitteeIndex)
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

	if err := vs.AttPool.AggregateUnaggregatedAttestations(); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not aggregate unaggregated attestations")
	}
	aggregatedAtts := vs.AttPool.AggregatedAttestationsBySlotIndex(req.Slot, req.CommitteeIndex)

	// Filter out the best aggregated attestation (ie. the one with the most aggregated bits).
	if len(aggregatedAtts) == 0 {
		aggregatedAtts = vs.AttPool.UnaggregatedAttestationsBySlotIndex(req.Slot, req.CommitteeIndex)
		if len(aggregatedAtts) == 0 {
			return nil, status.Errorf(codes.Internal, "Could not find attestation for slot and committee in pool")
		}
	}
	best := aggregatedAtts[0]
	for _, aggregatedAtt := range aggregatedAtts[1:] {
		if aggregatedAtt.AggregationBits.Count() > best.AggregationBits.Count() {
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
func (vs *Server) SubmitSignedAggregateSelectionProof(ctx context.Context, req *ethpb.SignedAggregateSubmitRequest) (*ethpb.SignedAggregateSubmitResponse, error) {
	if req.SignedAggregateAndProof == nil || req.SignedAggregateAndProof.Message == nil ||
		req.SignedAggregateAndProof.Message.Aggregate == nil || req.SignedAggregateAndProof.Message.Aggregate.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "Signed aggregate request can't be nil")
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

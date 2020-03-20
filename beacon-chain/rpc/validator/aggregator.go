package validator

import (
	"context"
	"github.com/sirupsen/logrus"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubmitAggregateSelectionProof is called by a validator when its assigned to be an aggregator.
// The aggregator submits the selection proof to obtain the aggregated attestation
// object to sign over.
func (as *Server) SubmitAggregateSelectionProof(ctx context.Context, req *ethpb.AggregateSelectionRequest) (*ethpb.AggregateSelectionResponse, error) {
	ctx, span := trace.StartSpan(ctx, "AggregatorServer.SubmitAggregateSelectionProof")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	if as.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	validatorIndex, exists, err := as.BeaconDB.ValidatorIndex(ctx, req.PublicKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get validator index from DB: %v", err)
	}
	if !exists {
		return nil, status.Error(codes.Internal, "Could not locate validator index in DB")
	}

	epoch := helpers.SlotToEpoch(req.Slot)
	activeValidatorIndices, err := as.HeadFetcher.HeadValidatorsIndices(epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get validators: %v", err)
	}
	seed, err := as.HeadFetcher.HeadSeed(epoch)
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

	// Retrieve the unaggregated attestation from pool.
	aggregatedAtts := as.AttPool.AggregatedAttestationsBySlotIndex(req.Slot, req.CommitteeIndex)

	// Filter out the best aggregated attestation (ie. the one with the most aggregated bits).
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
	return &ethpb.AggregateSelectionResponse{Aggregate: a}, nil
}

// SubmitSignedAggregateSelectionProof is called by a validator to broadcast a signed
// aggregated and proof object.
func (as *Server) SubmitSignedAggregateSelectionProof(ctx context.Context, req *ethpb.SignedAggregateSubmitRequest) (*ethpb.SignedAggregateSubmitResponse, error) {
	if err := as.P2P.Broadcast(ctx, req.Aggregate); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast signed aggregated attestation: %v", err)
	}

	log.WithFields(logrus.Fields{
		"slot":            req.Aggregate.Message.Aggregate.Data.Slot,
		"committeeIndex":  req.Aggregate.Message.Aggregate.Data.CommitteeIndex,
		"validatorIndex":  req.Aggregate.Message.AggregatorIndex,
		"aggregatedCount": req.Aggregate.Message.Aggregate.AggregationBits.Count(),
	}).Debug("Broadcasting aggregated attestation and proof")

	return &ethpb.SignedAggregateSubmitResponse{}, nil
}

package validator

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubmitAggregateAndProof is called by a validator when its assigned to be an aggregator.
// The beacon node will broadcast aggregated attestation and proof on the aggregator's behavior.
func (as *Server) SubmitAggregateAndProof(ctx context.Context, req *ethpb.AggregationRequest) (*ethpb.AggregationResponse, error) {
	ctx, span := trace.StartSpan(ctx, "AggregatorServer.SubmitAggregation")
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

	for _, aggregatedAtt := range aggregatedAtts {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if helpers.IsAggregated(aggregatedAtt) {
			if err := as.P2P.Broadcast(ctx, &ethpb.AggregateAttestationAndProof{
				AggregatorIndex: validatorIndex,
				SelectionProof:  req.SlotSignature,
				Aggregate:       aggregatedAtt,
			}); err != nil {
				return nil, status.Errorf(codes.Internal, "Could not broadcast aggregated attestation: %v", err)
			}

			log.WithFields(logrus.Fields{
				"slot":            req.Slot,
				"committeeIndex":  req.CommitteeIndex,
				"validatorIndex":  validatorIndex,
				"aggregatedCount": aggregatedAtt.AggregationBits.Count(),
			}).Debug("Broadcasting aggregated attestation and proof")
		}
	}

	return &ethpb.AggregationResponse{}, nil
}

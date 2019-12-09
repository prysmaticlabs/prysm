package aggregator

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log logrus.FieldLogger

func init() {
	log = logrus.WithField("prefix", "rpc/aggregator")
}

// Server defines a server implementation of the gRPC aggregator service.
type Server struct {
	BeaconDB    db.Database
	HeadFetcher blockchain.HeadFetcher
	SyncChecker sync.Checker
	AttPool     attestations.Pool
	P2p         p2p.Broadcaster
}

// SubmitAggregateAndProof is called by a validator when its assigned to be an aggregator.
// The beacon node will broadcast aggregated attestation and proof on the aggregator's behavior.
func (as *Server) SubmitAggregateAndProof(ctx context.Context, req *pb.AggregationRequest) (*pb.AggregationResponse, error) {
	ctx, span := trace.StartSpan(ctx, "AggregatorServer.SubmitAggregation")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	if as.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	headState, err := as.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head state: %v", err)
	}
	if req.Slot > headState.Slot {
		headState, err = state.ProcessSlots(ctx, headState, req.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", req.Slot, err)
		}
	}

	validatorIndex, exists, err := as.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(req.PublicKey))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get validator index from DB: %v", err)
	}
	if !exists {
		return nil, status.Error(codes.Internal, "Could not locate validator index in DB")
	}

	// Check if the validator is an aggregator
	sig, err := bls.SignatureFromBytes(req.SlotSignature)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert signature to byte: %v", err)
	}
	isAggregator, err := helpers.IsAggregator(headState, req.Slot, req.CommitteeIndex, sig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get aggregator status: %v", err)
	}
	if !isAggregator {
		return nil, status.Errorf(codes.InvalidArgument, "Validator is not an aggregator")
	}

	// Retrieve the aggregated attestation from pool
	atts := as.AttPool.UnaggregatedAttestations(req.Slot, req.CommitteeIndex)

	// Verify attestations are valid before aggregating and broadcasting them out.
	validAtts := make([]*ethpb.Attestation, 0, len(atts))
	for _, att := range atts {
		if err := blocks.VerifyAttestation(ctx, headState, att); err != nil {
			if err := as.AttPool.DeleteUnaggregatedAttestation(att); err != nil {
				return nil, status.Errorf(codes.Internal, "Could not delete invalid attestation: %v", err)
			}
			continue
		}
		validAtts = append(validAtts, att)
	}

	// Aggregate the attestations and broadcast them
	aggregatedAtts, err := helpers.AggregateAttestations(validAtts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not aggregate attestations: %v", err)
	}
	for _, att := range aggregatedAtts {
		if att.AggregationBits.Count() > 1 {
			if err := as.P2p.Broadcast(ctx, att); err != nil {
				return nil, status.Errorf(codes.Internal, "Could not broadcast aggregated attestation: %v", err)
			}
			if err := as.AttPool.SaveAggregatedAttestation(att); err != nil {
				return nil, status.Errorf(codes.Internal, "Could not save aggregated attestation: %v", err)
			}
		}
	}

	log.WithFields(logrus.Fields{
		"slot":           req.Slot,
		"validatorIndex": validatorIndex,
		"committeeIndex": req.CommitteeIndex,
	}).Debug("Broadcasting aggregated attestation and proof")

	return &pb.AggregationResponse{}, nil
}

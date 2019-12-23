package aggregator

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
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

	validatorIndex, exists, err := as.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(req.PublicKey))
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
	isAggregator, err := helpers.IsAggregator(uint64(len(committee)), req.Slot, req.CommitteeIndex, req.SlotSignature)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get aggregator status: %v", err)
	}
	if !isAggregator {
		return nil, status.Errorf(codes.InvalidArgument, "Validator is not an aggregator")
	}

	// Retrieve the unaggregated attestation from pool
	atts := as.AttPool.UnaggregatedAttestationsBySlotIndex(req.Slot, req.CommitteeIndex)

	// Aggregate the attestations and broadcast them.
	aggregatedAtts, err := helpers.AggregateAttestations(atts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not aggregate attestations: %v", err)
	}
	for _, aggregatedAtt := range aggregatedAtts {
		if helpers.IsAggregated(aggregatedAtt) {
			if err := as.P2p.Broadcast(ctx, &ethpb.AggregateAttestationAndProof{
				AggregatorIndex: validatorIndex,
				SelectionProof:  req.SlotSignature,
				Aggregate:       aggregatedAtt,
			}); err != nil {
				return nil, status.Errorf(codes.Internal, "Could not broadcast aggregated attestation: %v", err)
			}
			if err := as.AttPool.SaveAggregatedAttestation(aggregatedAtt); err != nil {
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

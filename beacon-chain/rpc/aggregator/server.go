package aggregator

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
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
}

// SubmitSlotSignature is called by a validator at every slot to check whether
// it's assigned to be an aggregator. If yes, server will broadcast aggregated attestation
// and proof on the validators behave.
func (as *Server) SubmitSlotSignature(ctx context.Context, req *pb.AggregationRequest) (*pb.AggregationResponse, error) {
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
	headState, err = state.ProcessSlots(ctx, headState, req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", req.Slot, err)
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

	validatorIndex, exists, err := as.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(req.PublicKey))
	if err != nil || !exists {
		return nil, status.Errorf(codes.Internal, "Could not get validator index from DB: %v", err)
	}

	// Broadcast aggregated attestation and proof if is an aggregator
	if isAggregator {
		log.WithFields(logrus.Fields{
			"slot":           req.Slot,
			"validatorIndex": validatorIndex,
			"committeeIndex": req.CommitteeIndex,
		}).Info("Broadcasting aggregated attestation and proof")

		return &pb.AggregationResponse{Aggregated: true}, nil
	}

	return &pb.AggregationResponse{Aggregated: false}, nil
}

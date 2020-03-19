package aggregator

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/validator"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"go.opencensus.io/trace"
)

// Server defines a server implementation of the gRPC aggregator service.
// Deprecated: Do not use.
type Server struct {
	ValidatorServer *validator.Server
}

// SubmitAggregateAndProof is called by a validator when its assigned to be an aggregator.
// The beacon node will broadcast aggregated attestation and proof on the aggregator's behavior.
// Deprecated: Use github.com/prysmaticlabs/prysm/beacon-chain/rpc/validator.SubmitAggregateAndProof.
// TODO(4952): Delete this method.
func (as *Server) SubmitAggregateAndProof(ctx context.Context, req *pb.AggregationRequest) (*pb.AggregationResponse, error) {
	ctx, span := trace.StartSpan(ctx, "AggregatorServer.SubmitAggregation")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	request := &ethpb.AggregationRequest{
		Slot:           req.Slot,
		CommitteeIndex: req.CommitteeIndex,
		PublicKey:      req.PublicKey,
		SlotSignature:  req.SlotSignature,
	}

	// Passthrough request to non-deprecated method.
	res, err := as.ValidatorServer.SubmitAggregateAndProof(ctx, request)
	if err != nil {
		return nil, err
	}
	return &pb.AggregationResponse{Root: res.AttestationDataRoot}, nil
}

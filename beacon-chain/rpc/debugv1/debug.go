package debugv1

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBeaconState returns the full beacon state for a given state id.
func (ds *Server) GetBeaconState(ctx context.Context, req *ethpb.StateRequest) (*ethpb.BeaconStateResponse, error) {
	state, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get state: %v", err)
	}

	protoState, err := state.ToProto()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not convert state to proto: %v", err)
	}

	return &ethpb.BeaconStateResponse{
		Data: protoState,
	}, nil
}

// ListForkChoiceHeads retrieves the fork choice leaves for the current head.
func (ds *Server) ListForkChoiceHeads(ctx context.Context, _ *ptypes.Empty) (*ethpb.ForkChoiceHeadsResponse, error) {
	return nil, errors.New("unimplemented")
}

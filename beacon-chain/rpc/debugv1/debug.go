package debugv1

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/statefetcher"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetBeaconState returns the full beacon state for a given state id.
func (ds *Server) GetBeaconState(ctx context.Context, req *ethpb.StateRequest) (*ethpb.BeaconStateResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetBeaconState")
	defer span.End()

	state, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		stateNotFoundErr, ok := err.(*statefetcher.StateNotFoundError)
		if ok {
			return nil, status.Errorf(codes.NotFound, "could not get state: %v", stateNotFoundErr)
		} else if errors.Is(err, statefetcher.ErrInvalidStateId) {
			return nil, status.Errorf(codes.InvalidArgument, "could not get state: %v", err)
		}
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
func (ds *Server) ListForkChoiceHeads(ctx context.Context, _ *emptypb.Empty) (*ethpb.ForkChoiceHeadsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "debugv1.ListForkChoiceHeads")
	defer span.End()

	headRoots, headSlots := ds.HeadFetcher.ChainHeads()
	resp := &ethpb.ForkChoiceHeadsResponse{
		Data: make([]*ethpb.ForkChoiceHead, len(headRoots)),
	}
	for i := range headRoots {
		resp.Data[i] = &ethpb.ForkChoiceHead{
			Root: headRoots[i][:],
			Slot: headSlots[i],
		}
	}

	return resp, nil
}

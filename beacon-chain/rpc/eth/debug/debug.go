package debug

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/statefetcher"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetBeaconState returns the full beacon state for a given state id.
func (ds *Server) GetBeaconState(ctx context.Context, req *ethpbv1.StateRequest) (*ethpbv1.BeaconStateResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetBeaconState")
	defer span.End()

	state, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		if stateNotFoundErr, ok := err.(*statefetcher.StateNotFoundError); ok {
			return nil, status.Errorf(codes.NotFound, "State not found: %v", stateNotFoundErr)
		} else if parseErr, ok := err.(*statefetcher.StateIdParseError); ok {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid state ID: %v", parseErr)
		}
		return nil, status.Errorf(codes.Internal, "Invalid state ID: %v", err)
	}

	protoState, err := state.ToProto()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert state to proto: %v", err)
	}

	return &ethpbv1.BeaconStateResponse{
		Data: protoState,
	}, nil
}

// GetBeaconStateSSZ returns the SSZ-serialized version of the full beacon state object for given stateId.
func (ds *Server) GetBeaconStateSSZ(ctx context.Context, req *ethpbv1.StateRequest) (*ethpbv1.BeaconStateSSZResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetBeaconStateSSZ")
	defer span.End()

	state, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		if stateNotFoundErr, ok := err.(*statefetcher.StateNotFoundError); ok {
			return nil, status.Errorf(codes.NotFound, "State not found: %v", stateNotFoundErr)
		} else if parseErr, ok := err.(*statefetcher.StateIdParseError); ok {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid state ID: %v", parseErr)
		}
		return nil, status.Errorf(codes.Internal, "Invalid state ID: %v", err)
	}

	sszState, err := state.MarshalSSZ()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal state into SSZ: %v", err)
	}

	return &ethpbv1.BeaconStateSSZResponse{Data: sszState}, nil
}

func (ds *Server) GetBeaconStateAltair(ctx context.Context, request *ethpbv2.StateRequestV2) (*ethpbv2.BeaconStateResponseV2, error) {
	panic("implement me")
}

func (ds *Server) GetBeaconStateSSZAltair(ctx context.Context, request *ethpbv2.StateRequestV2) (*ethpbv2.BeaconStateSSZResponseV2, error) {
	panic("implement me")
}

// ListForkChoiceHeads retrieves the fork choice leaves for the current head.
func (ds *Server) ListForkChoiceHeads(ctx context.Context, _ *emptypb.Empty) (*ethpbv1.ForkChoiceHeadsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "debug.ListForkChoiceHeads")
	defer span.End()

	headRoots, headSlots := ds.HeadFetcher.ChainHeads()
	resp := &ethpbv1.ForkChoiceHeadsResponse{
		Data: make([]*ethpbv1.ForkChoiceHead, len(headRoots)),
	}
	for i := range headRoots {
		resp.Data[i] = &ethpbv1.ForkChoiceHead{
			Root: headRoots[i][:],
			Slot: headSlots[i],
		}
	}

	return resp, nil
}

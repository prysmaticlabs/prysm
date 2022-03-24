package debug

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetBeaconState returns the full beacon state for a given state ID.
func (ds *Server) GetBeaconState(ctx context.Context, req *ethpbv1.StateRequest) (*ethpbv1.BeaconStateResponse, error) {
	ctx, span := trace.StartSpan(ctx, "debug.GetBeaconState")
	defer span.End()

	beaconSt, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	if beaconSt.Version() != version.Phase0 {
		return nil, status.Error(codes.Internal, "State has incorrect type")
	}
	protoSt, err := migration.BeaconStateToProto(beaconSt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert state to proto: %v", err)
	}

	return &ethpbv1.BeaconStateResponse{
		Data: protoSt,
	}, nil
}

// GetBeaconStateSSZ returns the SSZ-serialized version of the full beacon state object for given state ID.
func (ds *Server) GetBeaconStateSSZ(ctx context.Context, req *ethpbv1.StateRequest) (*ethpbv1.BeaconStateSSZResponse, error) {
	ctx, span := trace.StartSpan(ctx, "debug.GetBeaconStateSSZ")
	defer span.End()

	state, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	sszState, err := state.MarshalSSZ()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal state into SSZ: %v", err)
	}

	return &ethpbv1.BeaconStateSSZResponse{Data: sszState}, nil
}

// GetBeaconStateV2 returns the full beacon state for a given state ID.
func (ds *Server) GetBeaconStateV2(ctx context.Context, req *ethpbv2.StateRequestV2) (*ethpbv2.BeaconStateResponseV2, error) {
	ctx, span := trace.StartSpan(ctx, "debug.GetBeaconStateV2")
	defer span.End()

	beaconSt, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}
	switch beaconSt.Version() {
	case version.Phase0:
		protoSt, err := migration.BeaconStateToProto(beaconSt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not convert state to proto: %v", err)
		}
		return &ethpbv2.BeaconStateResponseV2{
			Version: ethpbv2.Version_PHASE0,
			Data: &ethpbv2.BeaconStateContainer{
				State: &ethpbv2.BeaconStateContainer_Phase0State{Phase0State: protoSt},
			},
		}, nil
	case version.Altair:
		altairState, ok := beaconSt.(state.BeaconStateAltair)
		if !ok {
			return nil, status.Error(codes.Internal, "Altair state type assertion failed")
		}
		protoState, err := migration.BeaconStateAltairToProto(altairState)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not convert state to proto: %v", err)
		}
		return &ethpbv2.BeaconStateResponseV2{
			Version: ethpbv2.Version_ALTAIR,
			Data: &ethpbv2.BeaconStateContainer{
				State: &ethpbv2.BeaconStateContainer_AltairState{AltairState: protoState},
			},
		}, nil
	case version.Bellatrix:
		bellatrixState, ok := beaconSt.(state.BeaconStateBellatrix)
		if !ok {
			return nil, status.Error(codes.Internal, "Bellatrix state type assertion failed")
		}
		protoState, err := migration.BeaconStateBellatrixToProto(bellatrixState)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not convert state to proto: %v", err)
		}
		return &ethpbv2.BeaconStateResponseV2{
			Version: ethpbv2.Version_BELLATRIX,
			Data: &ethpbv2.BeaconStateContainer{
				State: &ethpbv2.BeaconStateContainer_BellatrixState{BellatrixState: protoState},
			},
		}, nil
	default:
		return nil, status.Error(codes.Internal, "Unsupported state version")
	}
}

// GetBeaconStateSSZV2 returns the SSZ-serialized version of the full beacon state object for given state ID.
func (ds *Server) GetBeaconStateSSZV2(ctx context.Context, req *ethpbv2.StateRequestV2) (*ethpbv2.BeaconStateSSZResponseV2, error) {
	ctx, span := trace.StartSpan(ctx, "debug.GetBeaconStateSSZV2")
	defer span.End()

	state, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	sszState, err := state.MarshalSSZ()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal state into SSZ: %v", err)
	}

	return &ethpbv2.BeaconStateSSZResponseV2{Data: sszState}, nil
}

func (ds *Server) ListForkChoiceHeads(ctx context.Context, _ *emptypb.Empty) (*ethpbv1.ForkChoiceHeadsResponse, error) {
	_, span := trace.StartSpan(ctx, "debug.ListForkChoiceHeads")
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

// ListForkChoiceHeadsV2 retrieves the fork choice leaves for the current head.
func (ds *Server) ListForkChoiceHeadsV2(ctx context.Context, _ *emptypb.Empty) (*ethpbv2.ForkChoiceHeadsResponse, error) {
	_, span := trace.StartSpan(ctx, "debug.ListForkChoiceHeadsV2")
	defer span.End()

	headRoots, headSlots := ds.HeadFetcher.ChainHeads()
	resp := &ethpbv2.ForkChoiceHeadsResponse{
		Data: make([]*ethpbv2.ForkChoiceHead, len(headRoots)),
	}
	for i := range headRoots {
		isOptimistic, err := ds.HeadFetcher.IsOptimisticForRoot(ctx, headRoots[i])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not check if head is optimistic: %v", err)
		}
		resp.Data[i] = &ethpbv2.ForkChoiceHead{
			Root:                headRoots[i][:],
			Slot:                headSlots[i],
			ExecutionOptimistic: isOptimistic,
		}
	}

	return resp, nil
}

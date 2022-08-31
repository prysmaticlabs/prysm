package debug

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetBeaconStateSSZ returns the SSZ-serialized version of the full beacon state object for given state ID.
func (ds *Server) GetBeaconStateSSZ(ctx context.Context, req *ethpbv1.StateRequest) (*ethpbv2.SSZContainer, error) {
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

	return &ethpbv2.SSZContainer{Data: sszState}, nil
}

// GetBeaconStateV2 returns the full beacon state for a given state ID.
func (ds *Server) GetBeaconStateV2(ctx context.Context, req *ethpbv2.BeaconStateRequestV2) (*ethpbv2.BeaconStateResponseV2, error) {
	ctx, span := trace.StartSpan(ctx, "debug.GetBeaconStateV2")
	defer span.End()

	beaconSt, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}
	isOptimistic, err := helpers.IsOptimistic(ctx, beaconSt, ds.OptimisticModeFetcher)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
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
			ExecutionOptimistic: isOptimistic,
		}, nil
	case version.Altair:
		protoState, err := migration.BeaconStateAltairToProto(beaconSt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not convert state to proto: %v", err)
		}
		return &ethpbv2.BeaconStateResponseV2{
			Version: ethpbv2.Version_ALTAIR,
			Data: &ethpbv2.BeaconStateContainer{
				State: &ethpbv2.BeaconStateContainer_AltairState{AltairState: protoState},
			},
			ExecutionOptimistic: isOptimistic,
		}, nil
	case version.Bellatrix:
		protoState, err := migration.BeaconStateBellatrixToProto(beaconSt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not convert state to proto: %v", err)
		}
		return &ethpbv2.BeaconStateResponseV2{
			Version: ethpbv2.Version_BELLATRIX,
			Data: &ethpbv2.BeaconStateContainer{
				State: &ethpbv2.BeaconStateContainer_BellatrixState{BellatrixState: protoState},
			},
			ExecutionOptimistic: isOptimistic,
		}, nil
	default:
		return nil, status.Error(codes.Internal, "Unsupported state version")
	}
}

// GetBeaconStateSSZV2 returns the SSZ-serialized version of the full beacon state object for given state ID.
func (ds *Server) GetBeaconStateSSZV2(ctx context.Context, req *ethpbv2.BeaconStateRequestV2) (*ethpbv2.SSZContainer, error) {
	ctx, span := trace.StartSpan(ctx, "debug.GetBeaconStateSSZV2")
	defer span.End()

	st, err := ds.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	sszState, err := st.MarshalSSZ()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal state into SSZ: %v", err)
	}
	var ver ethpbv2.Version
	switch st.Version() {
	case version.Phase0:
		ver = ethpbv2.Version_PHASE0
	case version.Altair:
		ver = ethpbv2.Version_ALTAIR
	case version.Bellatrix:
		ver = ethpbv2.Version_BELLATRIX
	default:
		return nil, status.Error(codes.Internal, "Unsupported state version")
	}

	return &ethpbv2.SSZContainer{Data: sszState, Version: ver}, nil
}

// ListForkChoiceHeadsV2 retrieves the leaves of the current fork choice tree.
func (ds *Server) ListForkChoiceHeadsV2(ctx context.Context, _ *emptypb.Empty) (*ethpbv2.ForkChoiceHeadsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "debug.ListForkChoiceHeadsV2")
	defer span.End()

	headRoots, headSlots := ds.HeadFetcher.ChainHeads()
	resp := &ethpbv2.ForkChoiceHeadsResponse{
		Data: make([]*ethpbv2.ForkChoiceHead, len(headRoots)),
	}
	for i := range headRoots {
		isOptimistic, err := ds.OptimisticModeFetcher.IsOptimisticForRoot(ctx, headRoots[i])
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

// GetForkChoice returns a dump fork choice store.
func (ds *Server) GetForkChoice(ctx context.Context, _ *emptypb.Empty) (*ethpbv1.ForkChoiceResponse, error) {
	return ds.ForkFetcher.ForkChoicer().ForkChoiceDump(ctx)
}

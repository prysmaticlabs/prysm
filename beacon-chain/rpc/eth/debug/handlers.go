package debug

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"go.opencensus.io/trace"
)

// GetBeaconStateSSZ returns the SSZ-serialized version of the full beacon state object for given state ID.
//
// DEPRECATED: please use GetBeaconStateV2 instead
func (s *Server) GetBeaconStateSSZ(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "debug.GetBeaconStateSSZ")
	defer span.End()

	stateId := mux.Vars(r)["state_id"]
	if stateId == "" {
		http2.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}
	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}
	sszState, err := st.MarshalSSZ()
	if err != nil {
		http2.HandleError(w, "Could not marshal state into SSZ: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteSsz(w, sszState, "beacon_state.ssz")
}

// GetBeaconStateV2 returns the full beacon state for a given state ID.
func (s *Server) GetBeaconStateV2(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "debug.GetBeaconStateV2")
	defer span.End()

	if http2.SszRequested(r) {

	} else {

	}
}

// getBeaconStateV2 returns the JSON-serialized version of the full beacon state object for given state ID.
func (s *Server) getBeaconStateV2(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	stateId := mux.Vars(r)["state_id"]
	if stateId == "" {
		http2.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}
	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, []byte(stateId), s.OptimisticModeFetcher, s.Stater, s.ChainInfoFetcher, s.BeaconDB)
	if err != nil {
		http2.HandleError(w, "Could not check if state is optimistic: "+err.Error(), http.StatusInternalServerError)
		return
	}
	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		http2.HandleError(w, "Could not calculate root of latest block header: "+err.Error(), http.StatusInternalServerError)
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	switch st.Version() {
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
			Finalized:           isFinalized,
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
			Finalized:           isFinalized,
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
			Finalized:           isFinalized,
		}, nil
	case version.Capella:
		protoState, err := migration.BeaconStateCapellaToProto(beaconSt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not convert state to proto: %v", err)
		}
		return &ethpbv2.BeaconStateResponseV2{
			Version: ethpbv2.Version_CAPELLA,
			Data: &ethpbv2.BeaconStateContainer{
				State: &ethpbv2.BeaconStateContainer_CapellaState{CapellaState: protoState},
			},
			ExecutionOptimistic: isOptimistic,
			Finalized:           isFinalized,
		}, nil
	case version.Deneb:
		protoState, err := migration.BeaconStateDenebToProto(beaconSt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not convert state to proto: %v", err)
		}
		return &ethpbv2.BeaconStateResponseV2{
			Version: ethpbv2.Version_DENEB,
			Data: &ethpbv2.BeaconStateContainer{
				State: &ethpbv2.BeaconStateContainer_DenebState{DenebState: protoState},
			},
			ExecutionOptimistic: isOptimistic,
		}, nil
	default:
		http2.HandleError(w, "Unsupported state version", http.StatusInternalServerError)
	}
}

// getBeaconStateSSZV2 returns the SSZ-serialized version of the full beacon state object for given state ID.
func (ds *Server) getBeaconStateSSZV2(ctx context.Context, req *ethpbv2.BeaconStateRequestV2) (*ethpbv2.SSZContainer, error) {
	ctx, span := trace.StartSpan(ctx, "debug.GetBeaconStateSSZV2")
	defer span.End()

	st, err := ds.Stater.State(ctx, req.StateId)
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
	case version.Capella:
		ver = ethpbv2.Version_CAPELLA
	case version.Deneb:
		ver = ethpbv2.Version_DENEB
	default:
		return nil, status.Error(codes.Internal, "Unsupported state version")
	}

	return &ethpbv2.SSZContainer{Data: sszState, Version: ver}, nil
}

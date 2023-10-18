package debug

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v4/api"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
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

	stateId := mux.Vars(r)["state_id"]
	if stateId == "" {
		http2.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}

	if http2.SszRequested(r) {
		s.getBeaconStateSSZV2(ctx, w, []byte(stateId))
	} else {
		s.getBeaconStateV2(ctx, w, []byte(stateId))
	}
}

// getBeaconStateV2 returns the JSON-serialized version of the full beacon state object for given state ID.
func (s *Server) getBeaconStateV2(ctx context.Context, w http.ResponseWriter, id []byte) {
	st, err := s.Stater.State(ctx, id)
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, id, s.OptimisticModeFetcher, s.Stater, s.ChainInfoFetcher, s.BeaconDB)
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
	var respSt interface{}

	switch st.Version() {
	case version.Phase0:
		respSt, err = BeaconStateFromConsensus(st)
		if err != nil {
			http2.HandleError(w, "Could not convert consensus state to response : "+err.Error(), http.StatusInternalServerError)
			return
		}
	case version.Altair:
		respSt, err = BeaconStateAltairFromConsensus(st)
		if err != nil {
			http2.HandleError(w, "Could not convert consensus state to response : "+err.Error(), http.StatusInternalServerError)
			return
		}
	case version.Bellatrix:
		respSt, err = BeaconStateBellatrixFromConsensus(st)
		if err != nil {
			http2.HandleError(w, "Could not convert consensus state to response : "+err.Error(), http.StatusInternalServerError)
			return
		}
	case version.Capella:
		respSt, err = BeaconStateCapellaFromConsensus(st)
		if err != nil {
			http2.HandleError(w, "Could not convert consensus state to response : "+err.Error(), http.StatusInternalServerError)
			return
		}
	case version.Deneb:
		respSt, err = BeaconStateDenebFromConsensus(st)
		if err != nil {
			http2.HandleError(w, "Could not convert consensus state to response : "+err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http2.HandleError(w, "Unsupported state version", http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.Marshal(respSt)
	if err != nil {
		http2.HandleError(w, "Could not marshal state into JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
	ver := version.String(st.Version())
	resp := &GetBeaconStateV2Response{
		Version:             ver,
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
		Data:                jsonBytes,
	}
	w.Header().Set(api.VersionHeader, ver)
	http2.WriteJson(w, resp)
}

// getBeaconStateSSZV2 returns the SSZ-serialized version of the full beacon state object for given state ID.
func (s *Server) getBeaconStateSSZV2(ctx context.Context, w http.ResponseWriter, id []byte) {
	st, err := s.Stater.State(ctx, id)
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}
	sszState, err := st.MarshalSSZ()
	if err != nil {
		http2.HandleError(w, "Could not marshal state into SSZ: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(api.VersionHeader, version.String(st.Version()))
	http2.WriteSsz(w, sszState, "beacon_state.ssz")
}

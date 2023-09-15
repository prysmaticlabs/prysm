package lightclient

import (
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"go.opencensus.io/trace"
)

// GetLightClientBootstrap - implements https://github.com/ethereum/beacon-APIs/blob/263f4ed6c263c967f13279c7a9f5629b51c5fc55/apis/beacon/light_client/bootstrap.yaml
func (bs *Server) GetLightClientBootstrap(w http.ResponseWriter, req *http.Request) {
	// Prepare
	ctx, span := trace.StartSpan(req.Context(), "beacon.GetLightClientBootstrap")
	defer span.End()

	// Get the block
	blockRootParam, err := hexutil.Decode(mux.Vars(req)["block_root"])
	if err != nil {
		http2.HandleError(w, "invalid block root "+err.Error(), http.StatusBadRequest)
		return
	}

	var blockRoot [32]byte
	copy(blockRoot[:], blockRootParam)
	blk, err := bs.BeaconDB.Block(ctx, blockRoot)
	if !shared.WriteBlockFetchError(w, blk, err) {
		return
	}

	// Get the state
	state, err := bs.Stater.StateBySlot(ctx, blk.Block().Slot())
	if err != nil {
		http2.HandleError(w, "could not get state "+err.Error(), http.StatusInternalServerError)
		return
	}

	bootstrap, err := CreateLightClientBootstrap(ctx, state)
	if err != nil {
		http2.HandleError(w, "could not get light client bootstrap "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := &LightClientBootstrapResponse{
		Version: ethpbv2.Version(blk.Version()).String(),
		Data:    bootstrap,
	}

	http2.WriteJson(w, response)
}

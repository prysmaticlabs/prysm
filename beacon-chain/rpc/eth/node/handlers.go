package node

import (
	"net/http"
	"strconv"

	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	"go.opencensus.io/trace"
)

// GetSyncStatus requests the beacon node to describe if it's currently syncing or not, and
// if it is, what block it is up to.
func (s *Server) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "node.GetSyncStatus")
	defer span.End()

	isOptimistic, err := s.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		http2.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	headSlot := s.HeadFetcher.HeadSlot()
	response := &SyncStatusResponse{
		Data: &SyncStatusResponseData{
			HeadSlot:     strconv.FormatUint(uint64(headSlot), 10),
			SyncDistance: strconv.FormatUint(uint64(s.GenesisTimeFetcher.CurrentSlot()-headSlot), 10),
			IsSyncing:    s.SyncChecker.Syncing(),
			IsOptimistic: isOptimistic,
			ElOffline:    !s.ExecutionChainInfoFetcher.ExecutionClientConnected(),
		},
	}
	http2.WriteJson(w, response)
}

package node

import (
	"net/http"
	"strconv"
	"time"

	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	log "github.com/sirupsen/logrus"
)

// GetSyncStatus requests the beacon node to describe if it's currently syncing or not, and
// if it is, what block it is up to.
func (s *Server) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		http2.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Warnf("IsOptimistic: %s", time.Since(start))

	headSlot := s.HeadFetcher.HeadSlot()
	log.Warnf("HeadSlot: %s", time.Since(start))
	isSyncing := s.SyncChecker.Syncing()
	log.Warnf("Syncing: %s", time.Since(start))
	elOffline := !s.ExecutionChainInfoFetcher.ExecutionClientConnected()
	log.Warnf("ExecutionClientConnected: %s", time.Since(start))
	response := &SyncStatusResponse{
		Data: &SyncStatusResponseData{
			HeadSlot:     strconv.FormatUint(uint64(headSlot), 10),
			SyncDistance: strconv.FormatUint(uint64(s.GenesisTimeFetcher.CurrentSlot()-headSlot), 10),
			IsSyncing:    isSyncing,
			IsOptimistic: isOptimistic,
			ElOffline:    elOffline,
		},
	}
	http2.WriteJson(w, response)
}

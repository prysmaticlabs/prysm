package shared

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
)

func ValidateHex(w http.ResponseWriter, name string, s string) bool {
	if s == "" {
		errJson := &http2.DefaultErrorJson{
			Message: name + " is required",
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return false
	}
	if !bytesutil.IsHex([]byte(s)) {
		errJson := &http2.DefaultErrorJson{
			Message: name + " is invalid",
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return false
	}
	return true
}

func ValidateUint(w http.ResponseWriter, name string, s string) (uint64, bool) {
	if s == "" {
		errJson := &http2.DefaultErrorJson{
			Message: name + " is required",
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: name + " is invalid: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return 0, false
	}
	return v, true
}

// IsSyncing checks whether the beacon node is currently syncing and writes out the sync status.
func IsSyncing(
	ctx context.Context,
	w http.ResponseWriter,
	syncChecker sync.Checker,
	headFetcher blockchain.HeadFetcher,
	timeFetcher blockchain.TimeFetcher,
	optimisticModeFetcher blockchain.OptimisticModeFetcher,
) bool {
	if !syncChecker.Syncing() {
		return false
	}

	headSlot := headFetcher.HeadSlot()
	isOptimistic, err := optimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not check optimistic status: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return true
	}
	syncDetails := &SyncDetailsContainer{
		Data: &SyncDetails{
			HeadSlot:     strconv.FormatUint(uint64(headSlot), 10),
			SyncDistance: strconv.FormatUint(uint64(timeFetcher.CurrentSlot()-headSlot), 10),
			IsSyncing:    true,
			IsOptimistic: isOptimistic,
		},
	}

	msg := "Beacon node is currently syncing and not serving request on that endpoint"
	details, err := json.Marshal(syncDetails)
	if err == nil {
		msg += " Details: " + string(details)
	}
	errJson := &http2.DefaultErrorJson{
		Message: msg,
		Code:    http.StatusServiceUnavailable}
	http2.WriteError(w, errJson)
	return true
}

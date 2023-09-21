package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/sync"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
)

func UintFromQuery(w http.ResponseWriter, r *http.Request, name string) (bool, string, uint64) {
	raw := r.URL.Query().Get(name)
	if raw != "" {
		v, valid := ValidateUint(w, name, raw)
		if !valid {
			return false, "", 0
		}
		return true, raw, v
	}
	return true, "", 0
}

func ValidateHex(w http.ResponseWriter, name string, s string, length int) ([]byte, bool) {
	if s == "" {
		errJson := &http2.DefaultErrorJson{
			Message: name + " is required",
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return nil, false
	}
	hexBytes, err := hexutil.Decode(s)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: name + " is invalid: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return nil, false
	}
	if len(hexBytes) != length {
		http2.HandleError(w, fmt.Sprintf("Invalid %s: %s is not length %d", name, s, length), http.StatusBadRequest)
		return nil, false
	}
	return hexBytes, true
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

// IsOptimistic checks whether the beacon node is currently optimistic and writes it to the response.
func IsOptimistic(
	ctx context.Context,
	w http.ResponseWriter,
	optimisticModeFetcher blockchain.OptimisticModeFetcher,
) (bool, error) {
	isOptimistic, err := optimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not check optimistic status: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return true, err
	}
	if !isOptimistic {
		return false, nil
	}
	errJson := &http2.DefaultErrorJson{
		Code:    http.StatusServiceUnavailable,
		Message: "Beacon node is currently optimistic and not serving validators",
	}
	http2.WriteError(w, errJson)
	return true, nil
}

// DecodeHexWithLength takes a string and a length in bytes,
// and validates whether the string is a hex and has the correct length.
func DecodeHexWithLength(s string, length int) ([]byte, error) {
	bytes, err := hexutil.Decode(s)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("%s is not a valid hex", s))
	}
	if len(bytes) != length {
		return nil, fmt.Errorf("%s is not length %d bytes", s, length)
	}
	return bytes, nil
}

// DecodeHexWithMaxLength takes a string and a length in bytes,
// and validates whether the string is a hex and has the correct length.
func DecodeHexWithMaxLength(s string, maxLength int) ([]byte, error) {
	bytes, err := hexutil.Decode(s)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("%s is not a valid hex", s))
	}
	err = VerifyMaxLength(bytes, maxLength)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("length of %s exceeds max of %d bytes", s, maxLength))
	}
	return bytes, nil
}

// VerifyMaxLength takes a slice and a maximum length and validates the length.
func VerifyMaxLength[T any](v []T, max int) error {
	l := len(v)
	if l > max {
		return fmt.Errorf("length of %d exceeds max of %d", l, max)
	}
	return nil
}

package validator

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"

	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	statenative "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

type ValidatorCountResponse struct {
	ExecutionOptimistic string            `json:"execution_optimistic"`
	Finalized           string            `json:"finalized"`
	Data                []*ValidatorCount `json:"data"`
}

type ValidatorCount struct {
	Status string `json:"status"`
	Count  string `json:"count"`
}

// GetValidatorCount is a HTTP handler that serves the GET /eth/v1/beacon/states/{state_id}/validator_count endpoint.
// It returns the total validator count according to the given statuses provided as a query parameter.
//
// The state ID is expected to be a valid Beacon Chain state identifier.
// The status query parameter can be an array of strings with the following values: pending_initialized, pending_queued, active_ongoing,
// active_exiting, active_slashed, exited_unslashed, exited_slashed, withdrawal_possible, withdrawal_done, active, pending, exited, withdrawal.
// The response is a JSON object containing the total validator count for the specified statuses.
//
// Example usage:
//
//	GET /eth/v1/beacon/states/12345/validator_count?status=active&status=pending
//
// The above request will return a JSON response like:
//
//	{
//		"execution_optimistic": "false",
//		"finalized": "true",
//		"data": [
//			{
//				"status": "active",
//				"count": "13"
//			},
//			{
//				"status": "pending",
//				"count": "6"
//			}
//		]
//	}
func (vs *Server) GetValidatorCount(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetValidatorCount")
	defer span.End()

	stateID := mux.Vars(r)["state_id"]

	isOptimistic, err := helpers.IsOptimistic(ctx, []byte(stateID), vs.OptimisticModeFetcher, vs.Stater, vs.ChainInfoFetcher, vs.BeaconDB)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: fmt.Sprintf("could not check if slot's block is optimistic: %v", err),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	st, err := vs.Stater.State(ctx, []byte(stateID))
	if err != nil {
		var errJson *http2.DefaultErrorJson
		if _, ok := err.(*lookup.StateIdParseError); ok {
			errJson = &http2.DefaultErrorJson{
				Message: "invalid state ID",
				Code:    http.StatusBadRequest,
			}
		} else {
			errJson = &http2.DefaultErrorJson{
				Message: helpers.PrepareStateFetchError(err).Error(),
				Code:    http.StatusInternalServerError,
			}
		}
		http2.WriteError(w, errJson)
		return
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: fmt.Sprintf("could not calculate root of latest block header: %v", err),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	isFinalized := vs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	var statusVals []validator.ValidatorStatus
	for _, status := range r.URL.Query()["status"] {
		statusVal, ok := ethpb.ValidatorStatus_value[strings.ToUpper(status)]
		if !ok {
			errJson := &http2.DefaultErrorJson{
				Message: fmt.Sprintf("invalid status query parameter: %v", status),
				Code:    http.StatusBadRequest,
			}
			http2.WriteError(w, errJson)
			return
		}

		statusVals = append(statusVals, validator.ValidatorStatus(statusVal))
	}

	// If no status was provided then consider all the statuses to return validator count for each status.
	if len(statusVals) == 0 {
		for _, val := range ethpb.ValidatorStatus_value {
			statusVals = append(statusVals, validator.ValidatorStatus(val))
		}
	}

	epoch := slots.ToEpoch(st.Slot())
	valCount, err := validatorCountByStatus(st.Validators(), statusVals, epoch)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: fmt.Sprintf("could not get validator count: %v", err),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	valCountResponse := &ValidatorCountResponse{
		ExecutionOptimistic: strconv.FormatBool(isOptimistic),
		Finalized:           strconv.FormatBool(isFinalized),
		Data:                valCount,
	}

	http2.WriteJson(w, valCountResponse)
}

// validatorCountByStatus returns a slice of validator count for each status in the given epoch.
func validatorCountByStatus(validators []*eth.Validator, statuses []validator.ValidatorStatus, epoch primitives.Epoch) ([]*ValidatorCount, error) {
	countByStatus := make(map[validator.ValidatorStatus]uint64)
	for _, val := range validators {
		readOnlyVal, err := statenative.NewValidator(val)
		if err != nil {
			return nil, fmt.Errorf("could not convert validator: %v", err)
		}
		valStatus, err := helpers.ValidatorStatus(readOnlyVal, epoch)
		if err != nil {
			return nil, fmt.Errorf("could not get validator status: %v", err)
		}
		valSubStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
		if err != nil {
			return nil, fmt.Errorf("could not get validator sub status: %v", err)
		}

		for _, status := range statuses {
			if valStatus == status || valSubStatus == status {
				countByStatus[status]++
			}
		}
	}

	var resp []*ValidatorCount
	for status, count := range countByStatus {
		resp = append(resp, &ValidatorCount{
			Status: strings.ToLower(ethpb.ValidatorStatus_name[int32(status)]),
			Count:  strconv.FormatUint(count, 10),
		})
	}

	// Sort the response slice according to status strings for deterministic ordering of validator count response.
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Status < resp[j].Status
	})

	return resp, nil
}

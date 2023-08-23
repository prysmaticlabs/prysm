package validator

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	ExecutionOptimistic string          `json:"execution_optimistic"`
	Finalized           string          `json:"finalized"`
	Data                *ValidatorCount `json:"data"`
}

type ValidatorCount struct {
	ValidatorCount string `json:"validator_count"`
}

// GetValidatorCount is a HTTP handler that serves the GET /eth/v1/beacon/states/{state_id}/validator_count endpoint.
// It returns the total validator count according to the given status provided as a query parameter.
//
// The state ID is expected to be a valid Beacon Chain state identifier.
// The status query parameter should be one of the following values: pending_initialized, pending_queued, active_ongoing,
// active_exiting, active_slashed, exited_unslashed, exited_slashed, withdrawal_possible, withdrawal_done, active, pending, exited, withdrawal.
// The response is a JSON object containing the total validator count for the specified status.
//
// Example usage:
//
//	GET /eth/v1/beacon/states/12345/validator_count?status=active
//
// The above request will return a JSON response like:
//
//	{
//	 "execution_optimistic": "false",
//	 "finalized": "true",
//	 "data": {
//	  "validator_count": "1024"
//	 }
//	}
func (vs *Server) GetValidatorCount(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetValidatorCount")
	defer span.End()

	stateID := mux.Vars(r)["state_id"]

	isOptimistic, err := helpers.IsOptimistic(ctx, []byte(stateID), vs.OptimisticModeFetcher, vs.Stater, vs.ChainInfoFetcher, vs.BeaconDB)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: fmt.Errorf("could not check if slot's block is optimistic: %v", err).Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	st, err := vs.Stater.State(ctx, []byte(stateID))
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: helpers.PrepareStateFetchError(err).Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: fmt.Errorf("could not calculate root of latest block header: %v", err).Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	isFinalized := vs.FinalizationFetcher.IsFinalized(ctx, blockRoot)
	statusQuery := r.URL.Query().Get("status")
	statusVal, ok := ethpb.ValidatorStatus_value[strings.ToUpper(statusQuery)]
	if !ok {
		errJson := &http2.DefaultErrorJson{
			Message: fmt.Sprintf("invalid status query parameter: %v", statusVal),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	epoch := slots.ToEpoch(st.Slot())
	valCount, err := validatorCountByStatus(st.Validators(), validator.ValidatorStatus(statusVal), epoch)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: fmt.Errorf("could not get validator count: %v", err).Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	valCountResponse := &ValidatorCountResponse{
		ExecutionOptimistic: strconv.FormatBool(isOptimistic),
		Finalized:           strconv.FormatBool(isFinalized),
		Data: &ValidatorCount{
			ValidatorCount: strconv.FormatUint(valCount, 10),
		},
	}

	http2.WriteJson(w, valCountResponse)
}

func validatorCountByStatus(validators []*eth.Validator, status validator.ValidatorStatus, epoch primitives.Epoch) (uint64, error) {
	var resp uint64
	for _, val := range validators {
		readOnlyVal, err := statenative.NewValidator(val)
		if err != nil {
			return 0, fmt.Errorf("could not convert validator: %v", err)
		}
		valStatus, err := helpers.ValidatorStatus(readOnlyVal, epoch)
		if err != nil {
			return 0, fmt.Errorf("could not get validator status: %v", err)
		}
		valSubStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
		if err != nil {
			return 0, fmt.Errorf("could not get validator sub status: %v", err)
		}
		if valStatus == status || valSubStatus == status {
			resp++
		}
	}

	return resp, nil
}

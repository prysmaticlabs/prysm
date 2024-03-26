package validator

import (
	"encoding/json"
	"net/http"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// GetValidatorPerformance is an HTTP handler for GetValidatorPerformance.
func (s *Server) GetValidatorPerformance(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.GetValidatorPerformance")
	defer span.End()

	var req structs.GetValidatorPerformanceRequest
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handleHTTPError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	computed, err := s.CoreService.ComputeValidatorPerformance(
		ctx,
		&ethpb.ValidatorPerformanceRequest{
			PublicKeys: req.PublicKeys,
			Indices:    req.Indices,
		},
	)
	if err != nil {
		handleHTTPError(w, "Could not compute validator performance: "+err.Err.Error(), core.ErrorReasonToHTTP(err.Reason))
		return
	}
	response := &structs.GetValidatorPerformanceResponse{
		PublicKeys:                    computed.PublicKeys,
		CorrectlyVotedSource:          computed.CorrectlyVotedSource,
		CorrectlyVotedTarget:          computed.CorrectlyVotedTarget, // In altair, when this is true then the attestation was definitely included.
		CorrectlyVotedHead:            computed.CorrectlyVotedHead,
		CurrentEffectiveBalances:      computed.CurrentEffectiveBalances,
		BalancesBeforeEpochTransition: computed.BalancesBeforeEpochTransition,
		BalancesAfterEpochTransition:  computed.BalancesAfterEpochTransition,
		MissingValidators:             computed.MissingValidators,
		InactivityScores:              computed.InactivityScores, // Only populated in Altair
	}
	httputil.WriteJson(w, response)
}

func handleHTTPError(w http.ResponseWriter, message string, code int) {
	errJson := &httputil.DefaultJsonError{
		Message: message,
		Code:    code,
	}
	httputil.WriteError(w, errJson)
}

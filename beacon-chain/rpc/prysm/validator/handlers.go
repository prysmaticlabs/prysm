package validator

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"go.opencensus.io/trace"
)

// GetValidatorParticipation retrieves the validator participation information for a given epoch,
// it returns the information about validator's participation rate in voting on the proof of stake
// rules based on their balance compared to the total active validator balance.
func (s *Server) GetValidatorParticipation(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.GetValidatorParticipation")
	defer span.End()

	stateId := mux.Vars(r)["state_id"]
	if stateId == "" {
		httputil.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}

	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}
	stEpoch := slots.ToEpoch(st.Slot())
	vp, rpcError := s.CoreService.ValidatorParticipation(ctx, stEpoch)
	if rpcError != nil {
		httputil.HandleError(w, rpcError.Err.Error(), core.ErrorReasonToHTTP(rpcError.Reason))
		return
	}

	response := &structs.GetValidatorParticipationResponse{
		Epoch:     fmt.Sprintf("%d", vp.Epoch),
		Finalized: vp.Finalized,
		Participation: &structs.ValidatorParticipation{
			GlobalParticipationRate:          fmt.Sprintf("%f", vp.Participation.GlobalParticipationRate),
			VotedEther:                       fmt.Sprintf("%d", vp.Participation.VotedEther),
			EligibleEther:                    fmt.Sprintf("%d", vp.Participation.EligibleEther),
			CurrentEpochActiveGwei:           fmt.Sprintf("%d", vp.Participation.CurrentEpochActiveGwei),
			CurrentEpochAttestingGwei:        fmt.Sprintf("%d", vp.Participation.CurrentEpochAttestingGwei),
			CurrentEpochTargetAttestingGwei:  fmt.Sprintf("%d", vp.Participation.CurrentEpochTargetAttestingGwei),
			PreviousEpochActiveGwei:          fmt.Sprintf("%d", vp.Participation.PreviousEpochActiveGwei),
			PreviousEpochAttestingGwei:       fmt.Sprintf("%d", vp.Participation.PreviousEpochAttestingGwei),
			PreviousEpochTargetAttestingGwei: fmt.Sprintf("%d", vp.Participation.PreviousEpochTargetAttestingGwei),
			PreviousEpochHeadAttestingGwei:   fmt.Sprintf("%d", vp.Participation.PreviousEpochHeadAttestingGwei),
		},
	}
	httputil.WriteJson(w, response)
}

package validator

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
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

	stateId := strings.ReplaceAll(r.URL.Query().Get("state_id"), " ", "")
	var epoch uint64
	switch stateId {
	case "head":
		e, err := s.ChainInfoFetcher.ReceivedBlocksLastEpoch()
		if err != nil {
			httputil.HandleError(w, "Could not retrieve head root: "+err.Error(), http.StatusInternalServerError)
			return
		}
		epoch = e
	case "finalized":
		finalized := s.ChainInfoFetcher.FinalizedCheckpt()
		epoch = uint64(finalized.Epoch)
	case "genesis":
		epoch = 0
	default:
		_, e, ok := shared.UintFromQuery(w, r, "epoch", true)
		if !ok {
			currentSlot := s.CoreService.GenesisTimeFetcher.CurrentSlot()
			currentEpoch := slots.ToEpoch(currentSlot)
			epoch = uint64(currentEpoch)
		} else {
			epoch = e
		}
	}
	vp, err := s.CoreService.ValidatorParticipation(ctx, primitives.Epoch(epoch))
	if err != nil {
		httputil.HandleError(w, err.Err.Error(), core.ErrorReasonToHTTP(err.Reason))
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

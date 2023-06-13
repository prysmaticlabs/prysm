package beacon

import (
	"net/http"

	"github.com/prysmaticlabs/prysm/v4/network"
)

// BlockRewards is an HTTP handler for Beacon API getBlockRewards.
func (bs *Server) GetValidatorPerformance(w http.ResponseWriter, r *http.Request) {
	if bs.SyncChecker.Syncing() {
		errJson := &network.DefaultErrorJson{
			Message: "Syncing",
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	ctx := r.Context()
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get head state: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	currSlot := bs.GenesisTimeFetcher.CurrentSlot()
	_ = currSlot
	_ = headState
	response := &ValidatorPerformanceResponse{
		// PublicKeys:                    pubKeys,
		// CorrectlyVotedSource:          correctlyVotedSource,
		// CorrectlyVotedTarget:          correctlyVotedTarget, // In altair, when this is true then the attestation was definitely included.
		// CorrectlyVotedHead:            correctlyVotedHead,
		// CurrentEffectiveBalances:      effectiveBalances,
		// BalancesBeforeEpochTransition: beforeTransitionBalances,
		// BalancesAfterEpochTransition:  afterTransitionBalances,
		// MissingValidators:             missingValidators,
		// InactivityScores:              inactivityScores, // Only populated in Altair
	}
	network.WriteJson(w, response)
}

type ValidatorPerformanceResponse struct {
	// ProposerIndex     string `json:"proposer_index"`
	// Total             string `json:"total"`
	// Attestations      string `json:"attestations"`
	// SyncAggregate     string `json:"sync_aggregate"`
	// ProposerSlashings string `json:"proposer_slashings"`
	// AttesterSlashings string `json:"attester_slashings"`
}

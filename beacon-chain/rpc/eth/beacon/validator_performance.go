package beacon

import (
	"net/http"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/v1alpha1/beacon"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/network"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type ValidatorPerformanceRequest struct {
	PublicKeys [][]byte                    `json:"public_keys,omitempty"`
	Indices    []primitives.ValidatorIndex `json:"indices,omitempty"`
}

type ValidatorPerformanceResponse struct {
	PublicKeys                    [][]byte `json:"public_keys,omitempty"`
	CorrectlyVotedSource          []bool   `json:"correctly_voted_source,omitempty"`
	CorrectlyVotedTarget          []bool   `json:"correctly_voted_target,omitempty"`
	CorrectlyVotedHead            []bool   `json:"correctly_voted_head,omitempty"`
	CurrentEffectiveBalances      []uint64 `json:"current_effective_balances,omitempty"`
	BalancesBeforeEpochTransition []uint64 `json:"balances_before_epoch_transition,omitempty"`
	BalancesAfterEpochTransition  []uint64 `json:"balances_after_epoch_transition,omitempty"`
	MissingValidators             [][]byte `json:"missing_validators,omitempty"`
	InactivityScores              []uint64 `json:"inactivity_scores,omitempty"`
}

// GetValidatorPerformance is an HTTP handler for Beacon API GetValidatorPerformance.
func (bs *Server) GetValidatorPerformance(w http.ResponseWriter, r *http.Request) {
	if bs.SyncChecker.Syncing() {
		handleHTTPError(w, "Syncing", http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	currSlot := bs.GenesisTimeFetcher.CurrentSlot()
	computed, err := beacon.ComputeValidatorPerformance(
		ctx,
		&ethpb.ValidatorPerformanceRequest{
			PublicKeys: [][]byte{},
			Indices:    []uint64{},
		},
		bs.HeadFetcher,
		currSlot,
	)
	if err != nil {
		handleHTTPError(w, "Could not compute validator performance: "+err.Error(), http.StatusInternalServerError)
		return
	}
	response := &ValidatorPerformanceResponse{
		PublicKeys:                    computed.PublicKeys,
		CorrectlyVotedSource:          computed.CorrectlyVotedSource,
		CorrectlyVotedTarget:          computed.CorrectlyVotedTarget, // In altair, when this is true then the attestation was definitely included.
		CorrectlyVotedHead:            correctlyVotedHead,
		CurrentEffectiveBalances:      effectiveBalances,
		BalancesBeforeEpochTransition: beforeTransitionBalances,
		BalancesAfterEpochTransition:  afterTransitionBalances,
		MissingValidators:             missingValidators,
		InactivityScores:              inactivityScores, // Only populated in Altair
	}
	network.WriteJson(w, response)
}

func handleHTTPError(w http.ResponseWriter, message string, code int) {
	errJson := &network.DefaultErrorJson{
		Message: message,
		Code:    code,
	}
	network.WriteError(w, errJson)
}

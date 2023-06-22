package beacon

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	coreTime "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/network"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
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
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		handleHTTPError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	currSlot := bs.GenesisTimeFetcher.CurrentSlot()
	if currSlot > headState.Slot() {
		headRoot, err := bs.HeadFetcher.HeadRoot(ctx)
		if err != nil {
			handleHTTPError(w, "Could not retrieve head root: "+err.Error(), http.StatusInternalServerError)
			return
		}
		headState, err = transition.ProcessSlotsUsingNextSlotCache(ctx, headState, headRoot, currSlot)
		if err != nil {
			handleHTTPError(w, "Could not process slots up to "+strconv.FormatUint(uint64(currSlot), 10)+": "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	var validatorSummary []*precompute.Validator
	if headState.Version() == version.Phase0 {
		vp, bp, err := precompute.New(ctx, headState)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vp, bp, err = precompute.ProcessAttestations(ctx, headState, vp, bp)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		headState, err = precompute.ProcessRewardsAndPenaltiesPrecompute(headState, bp, vp, precompute.AttestationsDelta, precompute.ProposersDelta)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		validatorSummary = vp
	} else if headState.Version() >= version.Altair {
		vp, bp, err := altair.InitializePrecomputeValidators(ctx, headState)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vp, bp, err = altair.ProcessEpochParticipation(ctx, headState, bp, vp)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		headState, vp, err = altair.ProcessInactivityScores(ctx, headState, vp)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		headState, err = altair.ProcessRewardsAndPenaltiesPrecompute(headState, bp, vp)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		validatorSummary = vp
	} else {
		handleHTTPError(w, "Head state version "+strconv.Itoa(headState.Version())+" not supported", http.StatusInternalServerError)
		return
	}

	var req ValidatorPerformanceRequest
	if r.Body != http.NoBody {
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			handleHTTPError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	responseCap := len(req.Indices) + len(req.PublicKeys)
	validatorIndices := make([]primitives.ValidatorIndex, 0, responseCap)
	missingValidators := make([][]byte, 0, responseCap)

	filtered := map[primitives.ValidatorIndex]bool{} // Track filtered validators to prevent duplication in the response.
	// Convert the list of validator public keys to validator indices and add to the indices set.
	for _, pubKey := range req.PublicKeys {
		// Skip empty public key.
		if len(pubKey) == 0 {
			continue
		}
		pubkeyBytes := bytesutil.ToBytes48(pubKey)
		idx, ok := headState.ValidatorIndexByPubkey(pubkeyBytes)
		if !ok {
			// Validator index not found, track as missing.
			missingValidators = append(missingValidators, pubKey)
			continue
		}
		if !filtered[idx] {
			validatorIndices = append(validatorIndices, idx)
			filtered[idx] = true
		}
	}
	// Add provided indices to the indices set.
	for _, idx := range req.Indices {
		if !filtered[idx] {
			validatorIndices = append(validatorIndices, idx)
			filtered[idx] = true
		}
	}
	// Depending on the indices and public keys given, results might not be sorted.
	sort.Slice(validatorIndices, func(i, j int) bool {
		return validatorIndices[i] < validatorIndices[j]
	})

	currentEpoch := coreTime.CurrentEpoch(headState)
	responseCap = len(validatorIndices)
	pubKeys := make([][]byte, 0, responseCap)
	beforeTransitionBalances := make([]uint64, 0, responseCap)
	afterTransitionBalances := make([]uint64, 0, responseCap)
	effectiveBalances := make([]uint64, 0, responseCap)
	correctlyVotedSource := make([]bool, 0, responseCap)
	correctlyVotedTarget := make([]bool, 0, responseCap)
	correctlyVotedHead := make([]bool, 0, responseCap)
	inactivityScores := make([]uint64, 0, responseCap)
	// Append performance summaries.
	// Also track missing validators using public keys.
	for _, idx := range validatorIndices {
		val, err := headState.ValidatorAtIndexReadOnly(idx)
		if err != nil {
			handleHTTPError(w, "could not get validator: "+err.Error(), http.StatusInternalServerError)
			return
		}
		pubKey := val.PublicKey()
		if uint64(idx) >= uint64(len(validatorSummary)) {
			// Not listed in validator summary yet; treat it as missing.
			missingValidators = append(missingValidators, pubKey[:])
			continue
		}
		if !helpers.IsActiveValidatorUsingTrie(val, currentEpoch) {
			// Inactive validator; treat it as missing.
			missingValidators = append(missingValidators, pubKey[:])
			continue
		}

		summary := validatorSummary[idx]
		pubKeys = append(pubKeys, pubKey[:])
		effectiveBalances = append(effectiveBalances, summary.CurrentEpochEffectiveBalance)
		beforeTransitionBalances = append(beforeTransitionBalances, summary.BeforeEpochTransitionBalance)
		afterTransitionBalances = append(afterTransitionBalances, summary.AfterEpochTransitionBalance)
		correctlyVotedTarget = append(correctlyVotedTarget, summary.IsPrevEpochTargetAttester)
		correctlyVotedHead = append(correctlyVotedHead, summary.IsPrevEpochHeadAttester)

		if headState.Version() == version.Phase0 {
			correctlyVotedSource = append(correctlyVotedSource, summary.IsPrevEpochAttester)
		} else {
			correctlyVotedSource = append(correctlyVotedSource, summary.IsPrevEpochSourceAttester)
			inactivityScores = append(inactivityScores, summary.InactivityScore)
		}
	}

	response := &ValidatorPerformanceResponse{
		PublicKeys:                    pubKeys,
		CorrectlyVotedSource:          correctlyVotedSource,
		CorrectlyVotedTarget:          correctlyVotedTarget, // In altair, when this is true then the attestation was definitely included.
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

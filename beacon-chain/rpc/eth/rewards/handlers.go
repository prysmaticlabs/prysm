package rewards

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/altair"
	coreblocks "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"github.com/wealdtech/go-bytesutil"
)

// BlockRewards is an HTTP handler for Beacon API getBlockRewards.
func (s *Server) BlockRewards(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(r.URL.Path, "/")
	blockId := segments[len(segments)-1]

	blk, err := s.Blocker.Block(r.Context(), []byte(blockId))
	if errJson := handleGetBlockError(blk, err); errJson != nil {
		http2.WriteError(w, errJson)
		return
	}
	if blk.Version() == version.Phase0 {
		errJson := &http2.DefaultErrorJson{
			Message: "Block rewards are not supported for Phase 0 blocks",
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return
	}

	// We want to run several block processing functions that update the proposer's balance.
	// This will allow us to calculate proposer rewards for each operation (atts, slashings etc).
	// To do this, we replay the state up to the block's slot, but before processing the block.
	st, err := s.ReplayerBuilder.ReplayerForSlot(blk.Block().Slot()-1).ReplayToSlot(r.Context(), blk.Block().Slot())
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get state: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	proposerIndex := blk.Block().ProposerIndex()
	initBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	st, err = altair.ProcessAttestationsNoVerifySignature(r.Context(), st, blk)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get attestation rewards" + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	attBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	st, err = coreblocks.ProcessAttesterSlashings(r.Context(), st, blk.Block().Body().AttesterSlashings(), validators.SlashValidator)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get attester slashing rewards: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	attSlashingsBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	st, err = coreblocks.ProcessProposerSlashings(r.Context(), st, blk.Block().Body().ProposerSlashings(), validators.SlashValidator)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get proposer slashing rewards" + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	proposerSlashingsBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	sa, err := blk.Block().Body().SyncAggregate()
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get sync aggregate: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	var syncCommitteeReward uint64
	_, syncCommitteeReward, err = altair.ProcessSyncAggregate(r.Context(), st, sa)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get sync aggregate rewards: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get optimistic mode info: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get block root: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	response := &BlockRewardsResponse{
		Data: BlockRewards{
			ProposerIndex:     strconv.FormatUint(uint64(proposerIndex), 10),
			Total:             strconv.FormatUint(proposerSlashingsBalance-initBalance+syncCommitteeReward, 10),
			Attestations:      strconv.FormatUint(attBalance-initBalance, 10),
			SyncAggregate:     strconv.FormatUint(syncCommitteeReward, 10),
			ProposerSlashings: strconv.FormatUint(proposerSlashingsBalance-attSlashingsBalance, 10),
			AttesterSlashings: strconv.FormatUint(attSlashingsBalance-attBalance, 10),
		},
		ExecutionOptimistic: optimistic,
		Finalized:           s.FinalizationFetcher.IsFinalized(r.Context(), blkRoot),
	}
	http2.WriteJson(w, response)
}

// AttestationRewards retrieves attestation reward info for validators specified by array of public keys or validator index.
// If no array is provided, return reward info for every validator.
// TODO: Inclusion delay
func (s *Server) AttestationRewards(w http.ResponseWriter, r *http.Request) {
	st, ok := s.attRewardsState(w, r)
	if !ok {
		return
	}
	bal, vals, valIndices, ok := attRewardsBalancesAndVals(w, r, st)
	if !ok {
		return
	}
	totalRewards, ok := totalAttRewards(w, st, bal, vals, valIndices)
	if !ok {
		return
	}
	idealRewards, ok := idealAttRewards(w, st, bal, vals)
	if !ok {
		return
	}

	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get optimistic mode info: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	blkRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get block root: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	resp := &AttestationRewardsResponse{
		Data: AttestationRewards{
			IdealRewards: idealRewards,
			TotalRewards: totalRewards,
		},
		ExecutionOptimistic: optimistic,
		Finalized:           s.FinalizationFetcher.IsFinalized(r.Context(), blkRoot),
	}
	http2.WriteJson(w, resp)
}

// SyncCommitteeRewards retrieves rewards info for sync committee members specified by array of public keys or validator index.
// If no array is provided, return reward info for every committee member.
func (s *Server) SyncCommitteeRewards(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(r.URL.Path, "/")
	blockId := segments[len(segments)-1]

	blk, err := s.Blocker.Block(r.Context(), []byte(blockId))
	if errJson := handleGetBlockError(blk, err); errJson != nil {
		http2.WriteError(w, errJson)
		return
	}
	if blk.Version() == version.Phase0 {
		errJson := &http2.DefaultErrorJson{
			Message: "Sync committee rewards are not supported for Phase 0",
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return
	}
	st, err := s.ReplayerBuilder.ReplayerForSlot(blk.Block().Slot()-1).ReplayToSlot(r.Context(), blk.Block().Slot())
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get state: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	sa, err := blk.Block().Body().SyncAggregate()
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get sync aggregate: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	vals, valIndices, ok := syncRewardsVals(w, r, st)
	if !ok {
		return
	}
	preProcessBals := make([]uint64, len(vals))
	for i, valIdx := range valIndices {
		preProcessBals[i], err = st.BalanceAtIndex(valIdx)
		if err != nil {
			errJson := &http2.DefaultErrorJson{
				Message: "Could not get validator's balance: " + err.Error(),
				Code:    http.StatusInternalServerError,
			}
			http2.WriteError(w, errJson)
			return
		}
	}

	_, proposerReward, err := altair.ProcessSyncAggregate(r.Context(), st, sa)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get sync aggregate rewards: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	rewards := make([]int, len(preProcessBals))
	proposerIndex := blk.Block().ProposerIndex()
	for i, valIdx := range valIndices {
		bal, err := st.BalanceAtIndex(valIdx)
		if err != nil {
			errJson := &http2.DefaultErrorJson{
				Message: "Could not get validator's balance: " + err.Error(),
				Code:    http.StatusInternalServerError,
			}
			http2.WriteError(w, errJson)
			return
		}
		rewards[i] = int(bal - preProcessBals[i]) // lint:ignore uintcast
		if valIdx == proposerIndex {
			rewards[i] = rewards[i] - int(proposerReward) // lint:ignore uintcast
		}
	}

	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get optimistic mode info: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get block root: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	scRewards := make([]SyncCommitteeReward, len(valIndices))
	for i, valIdx := range valIndices {
		scRewards[i] = SyncCommitteeReward{
			ValidatorIndex: strconv.FormatUint(uint64(valIdx), 10),
			Reward:         strconv.Itoa(rewards[i]),
		}
	}
	response := &SyncCommitteeRewardsResponse{
		Data:                scRewards,
		ExecutionOptimistic: optimistic,
		Finalized:           s.FinalizationFetcher.IsFinalized(r.Context(), blkRoot),
	}
	http2.WriteJson(w, response)
}

func (s *Server) attRewardsState(w http.ResponseWriter, r *http.Request) (state.BeaconState, bool) {
	segments := strings.Split(r.URL.Path, "/")
	requestedEpoch, err := strconv.ParseUint(segments[len(segments)-1], 10, 64)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not decode epoch: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return nil, false
	}
	if primitives.Epoch(requestedEpoch) < params.BeaconConfig().AltairForkEpoch {
		errJson := &http2.DefaultErrorJson{
			Message: "Attestation rewards are not supported for Phase 0",
			Code:    http.StatusNotFound,
		}
		http2.WriteError(w, errJson)
		return nil, false
	}
	currentEpoch := uint64(slots.ToEpoch(s.TimeFetcher.CurrentSlot()))
	if requestedEpoch+1 >= currentEpoch {
		errJson := &http2.DefaultErrorJson{
			Code:    http.StatusNotFound,
			Message: "Attestation rewards are available after two epoch transitions to ensure all attestations have a chance of inclusion",
		}
		http2.WriteError(w, errJson)
		return nil, false
	}
	nextEpochEnd, err := slots.EpochEnd(primitives.Epoch(requestedEpoch + 1))
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get next epoch's ending slot: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return nil, false
	}
	st, err := s.Stater.StateBySlot(r.Context(), nextEpochEnd)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get state for epoch's starting slot: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return nil, false
	}
	return st, true
}

func attRewardsBalancesAndVals(
	w http.ResponseWriter,
	r *http.Request,
	st state.BeaconState,
) (*precompute.Balance, []*precompute.Validator, []primitives.ValidatorIndex, bool) {
	allVals, bal, err := altair.InitializePrecomputeValidators(r.Context(), st)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not initialize precompute validators: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return nil, nil, nil, false
	}
	allVals, bal, err = altair.ProcessEpochParticipation(r.Context(), st, bal, allVals)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not process epoch participation: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return nil, nil, nil, false
	}
	valIndices, ok := requestedValIndices(w, r, st, allVals)
	if !ok {
		return nil, nil, nil, false
	}
	if len(valIndices) == len(allVals) {
		return bal, allVals, valIndices, true
	} else {
		filteredVals := make([]*precompute.Validator, len(valIndices))
		for i, valIx := range valIndices {
			filteredVals[i] = allVals[valIx]
		}
		return bal, filteredVals, valIndices, true
	}
}

// idealAttRewards returns rewards for hypothetical, perfectly voting validators
// whose effective balances are over EJECTION_BALANCE and match balances in passed in validators.
func idealAttRewards(
	w http.ResponseWriter,
	st state.BeaconState,
	bal *precompute.Balance,
	vals []*precompute.Validator,
) ([]IdealAttestationReward, bool) {
	idealValsCount := uint64(16)
	minIdealBalance := uint64(17)
	maxIdealBalance := minIdealBalance + idealValsCount - 1
	idealRewards := make([]IdealAttestationReward, 0, idealValsCount)
	idealVals := make([]*precompute.Validator, 0, idealValsCount)
	increment := params.BeaconConfig().EffectiveBalanceIncrement
	for i := minIdealBalance; i <= maxIdealBalance; i++ {
		for _, v := range vals {
			if v.CurrentEpochEffectiveBalance/1e9 == i {
				effectiveBalance := i * increment
				idealVals = append(idealVals, &precompute.Validator{
					IsActivePrevEpoch:            true,
					IsSlashed:                    false,
					CurrentEpochEffectiveBalance: effectiveBalance,
					IsPrevEpochSourceAttester:    true,
					IsPrevEpochTargetAttester:    true,
					IsPrevEpochHeadAttester:      true,
				})
				idealRewards = append(idealRewards, IdealAttestationReward{EffectiveBalance: strconv.FormatUint(effectiveBalance, 10)})
				break
			}
		}
	}
	deltas, err := altair.AttestationsDelta(st, bal, idealVals)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get attestations delta: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return nil, false
	}
	for i, d := range deltas {
		idealRewards[i].Head = strconv.FormatUint(d.HeadReward, 10)
		if d.SourcePenalty > 0 {
			idealRewards[i].Source = fmt.Sprintf("-%s", strconv.FormatUint(d.SourcePenalty, 10))
		} else {
			idealRewards[i].Source = strconv.FormatUint(d.SourceReward, 10)
		}
		if d.TargetPenalty > 0 {
			idealRewards[i].Target = fmt.Sprintf("-%s", strconv.FormatUint(d.TargetPenalty, 10))
		} else {
			idealRewards[i].Target = strconv.FormatUint(d.TargetReward, 10)
		}
	}
	return idealRewards, true
}

func totalAttRewards(
	w http.ResponseWriter,
	st state.BeaconState,
	bal *precompute.Balance,
	vals []*precompute.Validator,
	valIndices []primitives.ValidatorIndex,
) ([]TotalAttestationReward, bool) {
	totalRewards := make([]TotalAttestationReward, len(valIndices))
	for i, v := range valIndices {
		totalRewards[i] = TotalAttestationReward{ValidatorIndex: strconv.FormatUint(uint64(v), 10)}
	}
	deltas, err := altair.AttestationsDelta(st, bal, vals)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get attestations delta: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return nil, false
	}
	for i, d := range deltas {
		totalRewards[i].Head = strconv.FormatUint(d.HeadReward, 10)
		if d.SourcePenalty > 0 {
			totalRewards[i].Source = fmt.Sprintf("-%s", strconv.FormatUint(d.SourcePenalty, 10))
		} else {
			totalRewards[i].Source = strconv.FormatUint(d.SourceReward, 10)
		}
		if d.TargetPenalty > 0 {
			totalRewards[i].Target = fmt.Sprintf("-%s", strconv.FormatUint(d.TargetPenalty, 10))
		} else {
			totalRewards[i].Target = strconv.FormatUint(d.TargetReward, 10)
		}
	}
	return totalRewards, true
}

func syncRewardsVals(
	w http.ResponseWriter,
	r *http.Request,
	st state.BeaconState,
) ([]*precompute.Validator, []primitives.ValidatorIndex, bool) {
	allVals, _, err := altair.InitializePrecomputeValidators(r.Context(), st)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not initialize precompute validators: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return nil, nil, false
	}
	valIndices, ok := requestedValIndices(w, r, st, allVals)
	if !ok {
		return nil, nil, false
	}

	sc, err := st.CurrentSyncCommittee()
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: "Could not get current sync committee: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return nil, nil, false
	}
	allScIndices := make([]primitives.ValidatorIndex, len(sc.Pubkeys))
	for i, pk := range sc.Pubkeys {
		valIdx, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pk))
		if !ok {
			errJson := &http2.DefaultErrorJson{
				Message: fmt.Sprintf("No validator index found for pubkey %#x", pk),
				Code:    http.StatusBadRequest,
			}
			http2.WriteError(w, errJson)
			return nil, nil, false
		}
		allScIndices[i] = valIdx
	}

	scIndices := make([]primitives.ValidatorIndex, 0, len(allScIndices))
	scVals := make([]*precompute.Validator, 0, len(allScIndices))
	for _, valIdx := range valIndices {
		for _, scIdx := range allScIndices {
			if valIdx == scIdx {
				scVals = append(scVals, allVals[valIdx])
				scIndices = append(scIndices, valIdx)
				break
			}
		}
	}

	return scVals, scIndices, true
}

func requestedValIndices(w http.ResponseWriter, r *http.Request, st state.BeaconState, allVals []*precompute.Validator) ([]primitives.ValidatorIndex, bool) {
	var rawValIds []string
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&rawValIds); err != nil {
			errJson := &http2.DefaultErrorJson{
				Message: "Could not decode validators: " + err.Error(),
				Code:    http.StatusBadRequest,
			}
			http2.WriteError(w, errJson)
			return nil, false
		}
	}
	valIndices := make([]primitives.ValidatorIndex, len(rawValIds))
	for i, v := range rawValIds {
		index, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			pubkey, err := bytesutil.FromHexString(v)
			if err != nil || len(pubkey) != fieldparams.BLSPubkeyLength {
				errJson := &http2.DefaultErrorJson{
					Message: fmt.Sprintf("%s is not a validator index or pubkey", v),
					Code:    http.StatusBadRequest,
				}
				http2.WriteError(w, errJson)
				return nil, false
			}
			var ok bool
			valIndices[i], ok = st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))
			if !ok {
				errJson := &http2.DefaultErrorJson{
					Message: fmt.Sprintf("No validator index found for pubkey %#x", pubkey),
					Code:    http.StatusBadRequest,
				}
				http2.WriteError(w, errJson)
				return nil, false
			}
		} else {
			if index >= uint64(st.NumValidators()) {
				errJson := &http2.DefaultErrorJson{
					Message: fmt.Sprintf("Validator index %d is too large. Maximum allowed index is %d", index, st.NumValidators()-1),
					Code:    http.StatusBadRequest,
				}
				http2.WriteError(w, errJson)
				return nil, false
			}
			valIndices[i] = primitives.ValidatorIndex(index)
		}
	}
	if len(valIndices) == 0 {
		valIndices = make([]primitives.ValidatorIndex, len(allVals))
		for i := 0; i < len(allVals); i++ {
			valIndices[i] = primitives.ValidatorIndex(i)
		}
	}

	return valIndices, true
}

func handleGetBlockError(blk interfaces.ReadOnlySignedBeaconBlock, err error) *http2.DefaultErrorJson {
	if errors.Is(err, lookup.BlockIdParseError{}) {
		return &http2.DefaultErrorJson{
			Message: "Invalid block ID: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
	}
	if err != nil {
		return &http2.DefaultErrorJson{
			Message: "Could not get block from block ID: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	if err := blocks.BeaconBlockIsNil(blk); err != nil {
		return &http2.DefaultErrorJson{
			Message: "Could not find requested block: " + err.Error(),
			Code:    http.StatusNotFound,
		}
	}
	return nil
}

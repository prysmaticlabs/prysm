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
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/math"
	"github.com/prysmaticlabs/prysm/v4/network"
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
		network.WriteError(w, errJson)
		return
	}
	if blk.Version() == version.Phase0 {
		errJson := &network.DefaultErrorJson{
			Message: "Block rewards are not supported for Phase 0 blocks",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}

	// We want to run several block processing functions that update the proposer's balance.
	// This will allow us to calculate proposer rewards for each operation (atts, slashings etc).
	// To do this, we replay the state up to the block's slot, but before processing the block.
	st, err := s.ReplayerBuilder.ReplayerForSlot(blk.Block().Slot()-1).ReplayToSlot(r.Context(), blk.Block().Slot())
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get state: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	proposerIndex := blk.Block().ProposerIndex()
	initBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	st, err = altair.ProcessAttestationsNoVerifySignature(r.Context(), st, blk)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get attestation rewards" + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	attBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	st, err = coreblocks.ProcessAttesterSlashings(r.Context(), st, blk.Block().Body().AttesterSlashings(), validators.SlashValidator)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get attester slashing rewards: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	attSlashingsBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	st, err = coreblocks.ProcessProposerSlashings(r.Context(), st, blk.Block().Body().ProposerSlashings(), validators.SlashValidator)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get proposer slashing rewards" + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	proposerSlashingsBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	sa, err := blk.Block().Body().SyncAggregate()
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get sync aggregate: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	var syncCommitteeReward uint64
	_, syncCommitteeReward, err = altair.ProcessSyncAggregate(r.Context(), st, sa)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get sync aggregate rewards: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get optimistic mode info: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get block root: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
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
	network.WriteJson(w, response)
}

// TODO: Explain the flow
func (s *Server) AttestationRewards(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(r.URL.Path, "/")
	epoch, err := strconv.ParseUint(segments[len(segments)-1], 10, 64)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not decode epoch: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}
	if primitives.Epoch(epoch) < params.BeaconConfig().AltairForkEpoch {
		errJson := &network.DefaultErrorJson{
			Message: "Attestation rewards are not supported for Phase 0",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}
	currentEpoch := uint64(slots.ToEpoch(s.TimeFetcher.CurrentSlot()))
	if epoch >= currentEpoch-1 {
		errJson := &network.DefaultErrorJson{
			Code:    http.StatusBadRequest,
			Message: "Attestation rewards are available after two epoch transitions to ensure all attestation have a chance of inclusion",
		}
		network.WriteError(w, errJson)
		return
	}

	epochStart, err := slots.EpochStart(primitives.Epoch(epoch + 2))
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get epoch's starting slot: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	st, err := s.Stater.StateBySlot(r.Context(), epochStart)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get state for epoch's starting slot: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	allVals, bal, err := altair.InitializePrecomputeValidators(r.Context(), st)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not initialize precompute validators: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}
	allVals, bal, err = altair.ProcessEpochParticipation(r.Context(), st, bal, allVals)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not process epoch participation: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}

	var rawValIds []string
	if r.Body != http.NoBody {
		if err = json.NewDecoder(r.Body).Decode(&rawValIds); err != nil {
			errJson := &network.DefaultErrorJson{
				Message: "Could not decode validators: " + err.Error(),
				Code:    http.StatusBadRequest,
			}
			network.WriteError(w, errJson)
			return
		}
	}
	valIndices := make([]primitives.ValidatorIndex, len(rawValIds))
	for i, v := range rawValIds {
		index, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			pubkey, err := bytesutil.FromHexString(v)
			if err != nil || len(pubkey) != fieldparams.BLSPubkeyLength {
				errJson := &network.DefaultErrorJson{
					Message: fmt.Sprintf("%s is not a validator index or pubkey", v),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
			var ok bool
			valIndices[i], ok = st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))
			if !ok {
				errJson := &network.DefaultErrorJson{
					Message: fmt.Sprintf("No validator index found for pubkey %#x", pubkey),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
		} else {
			if i >= st.NumValidators() {
				errJson := &network.DefaultErrorJson{
					Message: fmt.Sprintf("Validator index %d is too large. Maximum allowed index is %d", i, st.NumValidators()-1),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
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
	var filteredVals []*precompute.Validator
	if len(valIndices) == len(allVals) {
		filteredVals = allVals
	} else {
		filteredVals = make([]*precompute.Validator, len(valIndices))
		for i, valIx := range valIndices {
			filteredVals[i] = allVals[valIx]
		}
	}

	idealRewards := make([]AttReward, 32)
	idealVals := make([]*precompute.Validator, 32)
	increment := params.BeaconConfig().EffectiveBalanceIncrement
	for i := 1; i < 33; i++ {
		effectiveBalance := uint64(i) * increment
		idealVals[i-1] = &precompute.Validator{
			IsActivePrevEpoch:            true,
			IsSlashed:                    false,
			CurrentEpochEffectiveBalance: effectiveBalance,
			IsPrevEpochSourceAttester:    true,
			IsPrevEpochTargetAttester:    true,
			IsPrevEpochHeadAttester:      true,
		}
		idealRewards[i-1] = &IdealAttestationReward{EffectiveBalance: strconv.FormatUint(effectiveBalance, 10)}
	}
	err = attestationsDelta(idealRewards, st, bal, idealVals)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get attestations delta: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	totalRewards := make([]AttReward, len(valIndices))
	for i, v := range valIndices {
		totalRewards[i] = &TotalAttestationReward{ValidatorIndex: strconv.FormatUint(uint64(v), 10)}
	}
	err = attestationsDelta(totalRewards, st, bal, filteredVals)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not get attestations delta: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	ir := make([]IdealAttestationReward, len(idealRewards))
	for i, r := range idealRewards {
		ir[i] = *r.(*IdealAttestationReward)
	}
	tr := make([]TotalAttestationReward, len(totalRewards))
	for i, r := range totalRewards {
		tr[i] = *r.(*TotalAttestationReward)
	}
	resp := &AttestationRewardsResponse{
		Data: AttestationRewards{
			IdealRewards: ir,
			TotalRewards: tr,
		},
	}

	network.WriteJson(w, resp)
}

func attestationsDelta(
	rewards []AttReward,
	beaconState state.BeaconState,
	bal *precompute.Balance,
	vals []*precompute.Validator,
) error {
	cfg := params.BeaconConfig()
	prevEpoch := time.PrevEpoch(beaconState)
	finalizedEpoch := beaconState.FinalizedCheckpointEpoch()
	increment := cfg.EffectiveBalanceIncrement
	factor := cfg.BaseRewardFactor
	baseRewardMultiplier := increment * factor / math.CachedSquareRoot(bal.ActiveCurrentEpoch)
	leak := helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch)

	// Modified in Altair and Bellatrix.
	bias := cfg.InactivityScoreBias
	inactivityPenaltyQuotient, err := beaconState.InactivityPenaltyQuotient()
	if err != nil {
		return err
	}
	inactivityDenominator := bias * inactivityPenaltyQuotient

	for i, r := range rewards {
		err = attestationDelta(r, bal, vals[i], baseRewardMultiplier, inactivityDenominator, leak)
		if err != nil {
			return err
		}
	}

	return nil
}

func attestationDelta(
	reward AttReward,
	bal *precompute.Balance,
	val *precompute.Validator,
	baseRewardMultiplier, inactivityDenominator uint64,
	inactivityLeak bool) error {
	// TODO: Move this outside
	eligible := val.IsActivePrevEpoch || (val.IsSlashed && !val.IsWithdrawableCurrentEpoch)
	// Per spec `ActiveCurrentEpoch` can't be 0 to process attestation delta.
	if !eligible || bal.ActiveCurrentEpoch == 0 {
		return nil
	}

	cfg := params.BeaconConfig()
	increment := cfg.EffectiveBalanceIncrement
	effectiveBalance := val.CurrentEpochEffectiveBalance
	baseReward := (effectiveBalance / increment) * baseRewardMultiplier
	activeIncrement := bal.ActiveCurrentEpoch / increment

	weightDenominator := cfg.WeightDenominator
	srcWeight := cfg.TimelySourceWeight
	tgtWeight := cfg.TimelyTargetWeight
	headWeight := cfg.TimelyHeadWeight
	// Process source reward / penalty
	if val.IsPrevEpochSourceAttester && !val.IsSlashed {
		if inactivityLeak {
			reward.SetSource("0")
		} else {
			n := baseReward * srcWeight * (bal.PrevEpochAttested / increment)
			reward.SetSource(strconv.FormatUint(n/(activeIncrement*weightDenominator), 10))
		}
	} else {
		reward.SetSource(strconv.FormatUint(-(baseReward * srcWeight / weightDenominator), 10))
	}

	// Process target reward / penalty
	if val.IsPrevEpochTargetAttester && !val.IsSlashed {
		if inactivityLeak {
			reward.SetTarget("0")
		} else {
			n := baseReward * tgtWeight * (bal.PrevEpochTargetAttested / increment)
			reward.SetTarget(strconv.FormatUint(n/(activeIncrement*weightDenominator), 10))
		}
	} else {
		reward.SetTarget(strconv.FormatUint(-(baseReward * tgtWeight / weightDenominator), 10))
	}

	// Process head reward / penalty
	if val.IsPrevEpochHeadAttester && !val.IsSlashed {
		if inactivityLeak {
			reward.SetHead("0")
		} else {
			n := baseReward * headWeight * (bal.PrevEpochHeadAttested / increment)
			reward.SetHead(strconv.FormatUint(n/(activeIncrement*weightDenominator), 10))
		}
	} else {
		reward.SetHead("0")
	}

	// Process finality delay penalty
	// Apply an additional penalty to validators that did not vote on the correct target or slashed
	if !val.IsPrevEpochTargetAttester || val.IsSlashed {
		n, err := math.Mul64(effectiveBalance, val.InactivityScore)
		if err != nil {
			return err
		}
		r, err := strconv.ParseUint(reward.GetTarget(), 10, 64)
		// This should never happen because we set the target reward earlier in this function.
		if err != nil {
			return err
		}
		reward.SetTarget(strconv.FormatUint(r-n/inactivityDenominator, 10))
	}

	return nil
}

func handleGetBlockError(blk interfaces.ReadOnlySignedBeaconBlock, err error) *network.DefaultErrorJson {
	if errors.Is(err, lookup.BlockIdParseError{}) {
		return &network.DefaultErrorJson{
			Message: "Invalid block ID: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
	}
	if err != nil {
		return &network.DefaultErrorJson{
			Message: "Could not get block from block ID: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	if err := blocks.BeaconBlockIsNil(blk); err != nil {
		return &network.DefaultErrorJson{
			Message: "Could not find requested block" + err.Error(),
			Code:    http.StatusNotFound,
		}
	}
	return nil
}

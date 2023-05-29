package rewards

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/altair"
	coreblocks "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/network"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
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
		Data: &BlockRewards{
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
	var vals []string
	if err = json.NewDecoder(r.Body).Decode(&vals); err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not decode validators: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}

	currentEpoch := uint64(slots.ToEpoch(s.TimeFetcher.CurrentSlot()))
	if epoch > currentEpoch {
		errJson := &network.DefaultErrorJson{
			Message: fmt.Sprintf("Epoch cannot be in the future. Current epoch is %d", currentEpoch),
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}
	var st state.BeaconState
	if epoch == currentEpoch {
		st, err = s.HeadFetcher.HeadState(r.Context())
		if err != nil {
			errJson := &network.DefaultErrorJson{
				Message: "Could not get head state: " + err.Error(),
				Code:    http.StatusInternalServerError,
			}
			network.WriteError(w, errJson)
			return
		}
	} else {
		epochStart, err := slots.EpochStart(primitives.Epoch(epoch))
		if err != nil {
			errJson := &network.DefaultErrorJson{
				Message: "Could not get epoch's starting slot: " + err.Error(),
				Code:    http.StatusInternalServerError,
			}
			network.WriteError(w, errJson)
			return
		}
		st, err = s.Stater.StateBySlot(r.Context(), epochStart)
		if err != nil {
			errJson := &network.DefaultErrorJson{
				Message: "Could not get state for epoch's starting slot: " + err.Error(),
				Code:    http.StatusInternalServerError,
			}
			network.WriteError(w, errJson)
			return
		}
	}

	for _, v := range vals {
		isIndex := true
		index, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			isIndex = false
			pubkey, err := hexutil.Decode(v)
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: fmt.Sprintf("%s is not a validator index or pubkey", v),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
		}
		if isIndex {
			st.CurrentEpochParticipation()
		}
	}
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

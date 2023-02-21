package rewards

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	blocks2 "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/blockfetcher"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// BlockRewards is an HTTP handler for Beacon API getBlockRewards.
func (s *Server) BlockRewards(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(r.URL.Path, "/")
	blockId := segments[len(segments)-1]

	blk, err := s.BlockFetcher.Block(r.Context(), []byte(blockId))
	if errJson := handleGetBlockError(blk, err); errJson != nil {
		helpers.WriteError(w, errJson)
		return
	}
	if blk.Version() == version.Phase0 {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "block rewards are not supported for Phase 0 blocks").Error(),
			Code:    http.StatusBadRequest,
		}
		helpers.WriteError(w, errJson)
		return
	}
	st, err := s.CanonicalHistory.ReplayerForSlot(blk.Block().Slot()-1).ReplayToSlot(r.Context(), blk.Block().Slot())
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get state").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}

	proposerIndex := blk.Block().ProposerIndex()
	oldBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer's balance").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	st, err = altair.ProcessAttestationsNoVerifySignature(r.Context(), st, blk)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get attestation rewards").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	newBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer's balance").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	attsReward := newBalance - oldBalance
	oldBalance = newBalance
	st, err = blocks2.ProcessAttesterSlashings(r.Context(), st, blk.Block().Body().AttesterSlashings(), validators.SlashValidator)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get attester slashing rewards").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	newBalance, err = st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer's balance").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	attSlashingsReward := newBalance - oldBalance
	oldBalance = newBalance
	st, err = blocks2.ProcessProposerSlashings(r.Context(), st, blk.Block().Body().ProposerSlashings(), validators.SlashValidator)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer slashing rewards").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	newBalance, err = st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer's balance").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	proposerSlashingsReward := newBalance - oldBalance
	sa, err := blk.Block().Body().SyncAggregate()
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get sync aggregate").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	var syncCommitteeReward uint64
	st, syncCommitteeReward, err = altair.ProcessSyncAggregate(r.Context(), st, sa)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get sync aggregate rewards").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}

	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get optimistic mode info").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get block root").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}

	response := &BlockRewardsResponse{
		Data: &BlockRewards{
			ProposerIndex:     proposerIndex,
			Total:             attsReward + proposerSlashingsReward + attSlashingsReward + syncCommitteeReward,
			Attestations:      attsReward,
			SyncAggregate:     syncCommitteeReward,
			ProposerSlashings: proposerSlashingsReward,
			AttesterSlashings: attSlashingsReward,
		},
		ExecutionOptimistic: optimistic,
		Finalized:           s.FinalizationFetcher.IsFinalized(r.Context(), blkRoot),
	}
	helpers.WriteJson(w, response)
}

func handleGetBlockError(blk interfaces.ReadOnlySignedBeaconBlock, err error) *helpers.DefaultErrorJson {
	if errors.Is(err, blockfetcher.BlockIdParseError{}) {
		return &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "invalid block ID").Error(),
			Code:    http.StatusBadRequest,
		}
	}
	if err != nil {
		return &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get block from block ID").Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	if err := blocks.BeaconBlockIsNil(blk); err != nil {
		return &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not find requested block").Error(),
			Code:    http.StatusNotFound,
		}
	}
	return nil
}

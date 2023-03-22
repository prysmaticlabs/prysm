package rewards

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/altair"
	coreblocks "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/network"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
)

// BlockRewards is an HTTP handler for Beacon API getBlockRewards.
func (s *Server) BlockRewards(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(r.URL.Path, "/")
	blockId := segments[len(segments)-1]

	blk, err := s.BlockFetcher.Block(r.Context(), []byte(blockId))
	if errJson := handleGetBlockError(blk, err); errJson != nil {
		network.WriteError(w, errJson)
		return
	}
	if blk.Version() == version.Phase0 {
		errJson := &network.DefaultErrorJson{
			Message: "block rewards are not supported for Phase 0 blocks",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}
	st, err := s.ReplayerBuilder.ReplayerForSlot(blk.Block().Slot()-1).ReplayToSlot(r.Context(), blk.Block().Slot())
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get state").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	proposerIndex := blk.Block().ProposerIndex()
	oldBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer's balance").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	st, err = altair.ProcessAttestationsNoVerifySignature(r.Context(), st, blk)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get attestation rewards").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	newBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer's balance").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	attsReward := newBalance - oldBalance
	oldBalance = newBalance
	st, err = coreblocks.ProcessAttesterSlashings(r.Context(), st, blk.Block().Body().AttesterSlashings(), validators.SlashValidator)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get attester slashing rewards").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	newBalance, err = st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer's balance").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	attSlashingsReward := newBalance - oldBalance
	oldBalance = newBalance
	st, err = coreblocks.ProcessProposerSlashings(r.Context(), st, blk.Block().Body().ProposerSlashings(), validators.SlashValidator)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer slashing rewards").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	newBalance, err = st.BalanceAtIndex(proposerIndex)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get proposer's balance").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	proposerSlashingsReward := newBalance - oldBalance
	sa, err := blk.Block().Body().SyncAggregate()
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get sync aggregate").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	var syncCommitteeReward uint64
	st, syncCommitteeReward, err = altair.ProcessSyncAggregate(r.Context(), st, sa)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get sync aggregate rewards").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get optimistic mode info").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get block root").Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
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
	network.WriteJson(w, response)
}

func handleGetBlockError(blk interfaces.ReadOnlySignedBeaconBlock, err error) *network.DefaultErrorJson {
	if errors.Is(err, lookup.BlockIdParseError{}) {
		return &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "invalid block ID").Error(),
			Code:    http.StatusBadRequest,
		}
	}
	if err != nil {
		return &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get block from block ID").Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	if err := blocks.BeaconBlockIsNil(blk); err != nil {
		return &network.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not find requested block").Error(),
			Code:    http.StatusNotFound,
		}
	}
	return nil
}

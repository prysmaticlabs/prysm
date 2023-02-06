package rewards

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/blockfetcher"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

func (s *Server) BlockRewards(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not read request body").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	rewardsRequest := &BlockRewardsRequest{}
	if err := json.Unmarshal(body, rewardsRequest); err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not unmarshal request body").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	blk, err := s.BlockFetcher.Block(r.Context(), []byte(rewardsRequest.BlockId))
	if errJson := handleGetBlockError(blk, err); errJson != nil {
		helpers.WriteError(w, errJson)
		return
	}
	stateRoot := blk.Block().StateRoot()
	st, err := s.StateFetcher.State(r.Context(), stateRoot[:])
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not get state").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}
	vals, bal, err := altair.InitializePrecomputeValidators(r.Context(), st)
	if err != nil {
		errJson := &helpers.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not initialize precompute validators").Error(),
			Code:    http.StatusInternalServerError,
		}
		helpers.WriteError(w, errJson)
		return
	}

}

func handleGetBlockError(blk interfaces.SignedBeaconBlock, err error) *helpers.DefaultErrorJson {
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

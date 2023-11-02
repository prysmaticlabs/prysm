package rewards

import (
	"context"
	"net/http"
	"strconv"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/altair"
	coreblocks "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
)

// BlockRewardsFetcher is a interface that provides access to reward related responses
type BlockRewardsFetcher interface {
	GetBlockRewardsData(context.Context, interfaces.ReadOnlySignedBeaconBlock) (*BlockRewards, *http2.DefaultErrorJson)
	GetStateForRewards(context.Context, interfaces.ReadOnlySignedBeaconBlock) (state.BeaconState, *http2.DefaultErrorJson)
}

// BlockRewardService implements BlockRewardsFetcher and can be declared to access the underlying functions
type BlockRewardService struct {
	Replayer stategen.ReplayerBuilder
}

// GetBlockRewardsData returns the BlockRewards Object which is used for the BlockRewardsResponse and ProduceBlockV3
func (rs *BlockRewardService) GetBlockRewardsData(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*BlockRewards, *http2.DefaultErrorJson) {
	st, httpErr := rs.GetStateForRewards(ctx, blk)
	if httpErr != nil {
		return nil, httpErr
	}

	proposerIndex := blk.Block().ProposerIndex()
	initBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	st, err = altair.ProcessAttestationsNoVerifySignature(ctx, st, blk)
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get attestation rewards: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	attBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	st, err = coreblocks.ProcessAttesterSlashings(ctx, st, blk.Block().Body().AttesterSlashings(), validators.SlashValidator)
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get attester slashing rewards: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	attSlashingsBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	st, err = coreblocks.ProcessProposerSlashings(ctx, st, blk.Block().Body().ProposerSlashings(), validators.SlashValidator)
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get proposer slashing rewards: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	proposerSlashingsBalance, err := st.BalanceAtIndex(proposerIndex)
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get proposer's balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	sa, err := blk.Block().Body().SyncAggregate()
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get sync aggregate: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	var syncCommitteeReward uint64
	_, syncCommitteeReward, err = altair.ProcessSyncAggregate(ctx, st, sa)
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get sync aggregate rewards: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}

	return &BlockRewards{
		ProposerIndex:     strconv.FormatUint(uint64(proposerIndex), 10),
		Total:             strconv.FormatUint(proposerSlashingsBalance-initBalance+syncCommitteeReward, 10),
		Attestations:      strconv.FormatUint(attBalance-initBalance, 10),
		SyncAggregate:     strconv.FormatUint(syncCommitteeReward, 10),
		ProposerSlashings: strconv.FormatUint(proposerSlashingsBalance-attSlashingsBalance, 10),
		AttesterSlashings: strconv.FormatUint(attSlashingsBalance-attBalance, 10),
	}, nil
}

// GetStateForRewards returns the state replayed up to the block's slot
func (rs *BlockRewardService) GetStateForRewards(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (state.BeaconState, *http2.DefaultErrorJson) {
	// We want to run several block processing functions that update the proposer's balance.
	// This will allow us to calculate proposer rewards for each operation (atts, slashings etc).
	// To do this, we replay the state up to the block's slot, but before processing the block.
	st, err := rs.Replayer.ReplayerForSlot(blk.Block().Slot()-1).ReplayToSlot(ctx, blk.Block().Slot())
	if err != nil {
		return nil, &http2.DefaultErrorJson{
			Message: "Could not get state: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	return st, nil
}

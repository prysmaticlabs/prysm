package rewards

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
)

// BlockRewardsFetcher retrieves the Consensus Payload ( aka block rewards) of the passed in block
type BlockRewardsFetcher interface {
	GetBlockRewardsData(context.Context, interfaces.ReadOnlySignedBeaconBlock) (*BlockRewards, *http2.DefaultErrorJson)
	GetStateForRewards(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (state.BeaconState, *http2.DefaultErrorJson)
}

type BlockRewardService struct {
	Replayer stategen.ReplayerBuilder
}

type BlockRewardsResponse struct {
	Data                *BlockRewards `json:"data"`
	ExecutionOptimistic bool          `json:"execution_optimistic"`
	Finalized           bool          `json:"finalized"`
}

type BlockRewards struct {
	ProposerIndex     string `json:"proposer_index"`
	Total             string `json:"total"`
	Attestations      string `json:"attestations"`
	SyncAggregate     string `json:"sync_aggregate"`
	ProposerSlashings string `json:"proposer_slashings"`
	AttesterSlashings string `json:"attester_slashings"`
}

type AttestationRewardsResponse struct {
	Data                AttestationRewards `json:"data"`
	ExecutionOptimistic bool               `json:"execution_optimistic"`
	Finalized           bool               `json:"finalized"`
}

type AttestationRewards struct {
	IdealRewards []IdealAttestationReward `json:"ideal_rewards"`
	TotalRewards []TotalAttestationReward `json:"total_rewards"`
}

type IdealAttestationReward struct {
	EffectiveBalance string `json:"effective_balance"`
	Head             string `json:"head"`
	Target           string `json:"target"`
	Source           string `json:"source"`
}

type TotalAttestationReward struct {
	ValidatorIndex string `json:"validator_index"`
	Head           string `json:"head"`
	Target         string `json:"target"`
	Source         string `json:"source"`
	InclusionDelay string `json:"inclusion_delay"`
}

type SyncCommitteeRewardsResponse struct {
	Data                []SyncCommitteeReward `json:"data"`
	ExecutionOptimistic bool                  `json:"execution_optimistic"`
	Finalized           bool                  `json:"finalized"`
}

type SyncCommitteeReward struct {
	ValidatorIndex string `json:"validator_index"`
	Reward         string `json:"reward"`
}

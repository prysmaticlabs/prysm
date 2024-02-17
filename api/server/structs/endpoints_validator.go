package structs

import (
	"encoding/json"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

type AggregateAttestationResponse struct {
	Data *Attestation `json:"data"`
}

type SubmitContributionAndProofsRequest struct {
	Data []*SignedContributionAndProof `json:"data"`
}

type SubmitAggregateAndProofsRequest struct {
	Data []*SignedAggregateAttestationAndProof `json:"data"`
}

type SubmitSyncCommitteeSubscriptionsRequest struct {
	Data []*SyncCommitteeSubscription `json:"data"`
}

type SubmitBeaconCommitteeSubscriptionsRequest struct {
	Data []*BeaconCommitteeSubscription `json:"data"`
}

type GetAttestationDataResponse struct {
	Data *AttestationData `json:"data"`
}

type ProduceSyncCommitteeContributionResponse struct {
	Data *SyncCommitteeContribution `json:"data"`
}

type GetAttesterDutiesResponse struct {
	DependentRoot       string          `json:"dependent_root"`
	ExecutionOptimistic bool            `json:"execution_optimistic"`
	Data                []*AttesterDuty `json:"data"`
}

type AttesterDuty struct {
	Pubkey                  string `json:"pubkey"`
	ValidatorIndex          string `json:"validator_index"`
	CommitteeIndex          string `json:"committee_index"`
	CommitteeLength         string `json:"committee_length"`
	CommitteesAtSlot        string `json:"committees_at_slot"`
	ValidatorCommitteeIndex string `json:"validator_committee_index"`
	Slot                    string `json:"slot"`
}

type GetProposerDutiesResponse struct {
	DependentRoot       string          `json:"dependent_root"`
	ExecutionOptimistic bool            `json:"execution_optimistic"`
	Data                []*ProposerDuty `json:"data"`
}

type ProposerDuty struct {
	Pubkey         string `json:"pubkey"`
	ValidatorIndex string `json:"validator_index"`
	Slot           string `json:"slot"`
}

type GetSyncCommitteeDutiesResponse struct {
	ExecutionOptimistic bool                 `json:"execution_optimistic"`
	Data                []*SyncCommitteeDuty `json:"data"`
}

type SyncCommitteeDuty struct {
	Pubkey                        string   `json:"pubkey"`
	ValidatorIndex                string   `json:"validator_index"`
	ValidatorSyncCommitteeIndices []string `json:"validator_sync_committee_indices"`
}

// ProduceBlockV3Response is a wrapper json object for the returned block from the ProduceBlockV3 endpoint
type ProduceBlockV3Response struct {
	Version                 string          `json:"version"`
	ExecutionPayloadBlinded bool            `json:"execution_payload_blinded"`
	ExecutionPayloadValue   string          `json:"execution_payload_value"`
	ConsensusBlockValue     string          `json:"consensus_block_value"`
	Data                    json.RawMessage `json:"data"` // represents the block values based on the version
}

type GetLivenessResponse struct {
	Data []*Liveness `json:"data"`
}

type Liveness struct {
	Index  string `json:"index"`
	IsLive bool   `json:"is_live"`
}

type GetValidatorCountResponse struct {
	ExecutionOptimistic string            `json:"execution_optimistic"`
	Finalized           string            `json:"finalized"`
	Data                []*ValidatorCount `json:"data"`
}

type ValidatorCount struct {
	Status string `json:"status"`
	Count  string `json:"count"`
}

type GetValidatorPerformanceRequest struct {
	PublicKeys [][]byte                    `json:"public_keys,omitempty"`
	Indices    []primitives.ValidatorIndex `json:"indices,omitempty"`
}

type GetValidatorPerformanceResponse struct {
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

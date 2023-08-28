package validator

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
)

type AggregateAttestationResponse struct {
	Data *shared.Attestation `json:"data"`
}

type SubmitContributionAndProofsRequest struct {
	Data []*shared.SignedContributionAndProof `json:"data" validate:"required,dive"`
}

type SubmitAggregateAndProofsRequest struct {
	Data []*shared.SignedAggregateAttestationAndProof `json:"data" validate:"required,dive"`
}

type SubmitSyncCommitteeSubscriptionsRequest struct {
	Data []*shared.SyncCommitteeSubscription `json:"data" validate:"required,dive"`
}

type SubmitBeaconCommitteeSubscriptionsRequest struct {
	Data []*shared.BeaconCommitteeSubscription `json:"data" validate:"required,dive"`
}

type GetAttestationDataResponse struct {
	Data *shared.AttestationData `json:"data"`
}

type ProduceSyncCommitteeContributionResponse struct {
	Data *shared.SyncCommitteeContribution `json:"data"`
}

type GetAttesterDutiesRequest struct {
	ValidatorIndices []string `json:"validator_indices"`
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

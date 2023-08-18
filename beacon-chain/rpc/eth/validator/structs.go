package validator

import "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"

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

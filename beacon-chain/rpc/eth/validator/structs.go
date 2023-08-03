package validator

import "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"

type AggregateAttestationResponse struct {
	Data *shared.Attestation `json:"data" validate:"required"`
}

type SubmitContributionAndProofsRequest struct {
	Data []*shared.SignedContributionAndProof `json:"data" validate:"required"`
}

type SubmitAggregateAndProofsRequest struct {
	Data []*shared.SignedAggregateAttestationAndProof `json:"data" validate:"required"`
}

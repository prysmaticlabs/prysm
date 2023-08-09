package validator

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

type AggregateAttestationResponse struct {
	Data *shared.Attestation `json:"data" validate:"required"`
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

type ProduceSyncCommitteeContributionRequest struct {
	Slot              primitives.Slot `json:"slot,omitempty"`
	SubcommitteeIndex uint64          `json:"subcommittee_index,omitempty"`
	BeaconBlockRoot   []byte          `json:"beacon_block_root,omitempty"`
}

type ProduceSyncCommitteeContributionResponse struct {
	Data *shared.SyncCommitteeContribution
}

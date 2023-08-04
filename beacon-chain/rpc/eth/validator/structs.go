package validator

import (
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

type AggregateAttestationResponse struct {
	Data *shared.Attestation `json:"data" validate:"required"`
}

type SubmitContributionAndProofsRequest struct {
	Data []*shared.SignedContributionAndProof `json:"data" validate:"required"`
}

type SubmitAggregateAndProofsRequest struct {
	Data []*shared.SignedAggregateAttestationAndProof `json:"data" validate:"required"`
}

type ProduceSyncCommitteeContributionRequest struct {
	Slot              primitives.Slot `json:"slot,omitempty"`
	SubcommitteeIndex uint64          `json:"subcommittee_index,omitempty"`
	BeaconBlockRoot   []byte          `json:"beacon_block_root,omitempty"`
}

type ProduceSyncCommitteeContributionResponse struct {
	Data *SyncCommitteeContribution
}

type SyncCommitteeContribution struct {
	Slot              primitives.Slot       `json:"slot,omitempty"`
	BeaconBlockRoot   []byte                `json:"beacon_block_root,omitempty"`
	SubcommitteeIndex uint64                `json:"subcommittee_index,omitempty"`
	AggregationBits   bitfield.Bitvector128 `json:"aggregation_bits,omitempty"`
	Signature         []byte                `json:"signature,omitempty"`
}

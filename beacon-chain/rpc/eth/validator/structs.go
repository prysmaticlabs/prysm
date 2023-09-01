package validator

import (
	"encoding/json"

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

// ProduceBlockV3Response is a wrapper json object for the returned block from the ProduceBlockV3 endpoint
type ProduceBlockV3Response struct {
	Version                 string          `json:"version"`
	ExecutionPayloadBlinded bool            `json:"execution_payload_blinded"`
	ExecutionPayloadValue   string          `json:"execution_payload_value"`
	Data                    json.RawMessage `json:"data"` // represents the block values based on the version
}

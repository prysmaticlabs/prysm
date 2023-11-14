package events

import (
	"encoding/json"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
)

type HeadEvent struct {
	Slot                      string `json:"slot"`
	Block                     string `json:"block" hex:"true"`
	State                     string `json:"state" hex:"true"`
	EpochTransition           bool   `json:"epoch_transition"`
	ExecutionOptimistic       bool   `json:"execution_optimistic"`
	PreviousDutyDependentRoot string `json:"previous_duty_dependent_root" hex:"true"`
	CurrentDutyDependentRoot  string `json:"current_duty_dependent_root" hex:"true"`
}

type BlockEvent struct {
	Slot                string `json:"slot"`
	Block               string `json:"block" hex:"true"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type AggregatedAttEventSource struct {
	Aggregate *shared.Attestation `json:"aggregate"`
}

type UnaggregatedAttEventSource struct {
	AggregationBits string                  `json:"aggregation_bits" hex:"true"`
	Data            *shared.AttestationData `json:"data"`
	Signature       string                  `json:"signature" hex:"true"`
}

type FinalizedCheckpointEvent struct {
	Block               string `json:"block" hex:"true"`
	State               string `json:"state" hex:"true"`
	Epoch               string `json:"epoch"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type ChainReorgEvent struct {
	Slot                string `json:"slot"`
	Depth               string `json:"depth"`
	OldHeadBlock        string `json:"old_head_block" hex:"true"`
	NewHeadBlock        string `json:"old_head_state" hex:"true"`
	OldHeadState        string `json:"new_head_block" hex:"true"`
	NewHeadState        string `json:"new_head_state" hex:"true"`
	Epoch               string `json:"epoch"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type PayloadAttributesEvent struct {
	Version string          `json:"version"`
	Data    json.RawMessage `json:"data"`
}

type PayloadAttributesEventData struct {
	ProposerIndex     string          `json:"proposer_index"`
	ProposalSlot      string          `json:"proposal_slot"`
	ParentBlockNumber string          `json:"parent_block_number"`
	ParentBlockRoot   string          `json:"parent_block_root" hex:"true"`
	ParentBlockHash   string          `json:"parent_block_hash" hex:"true"`
	PayloadAttributes json.RawMessage `json:"payload_attributes"`
}

type PayloadAttributesV1 struct {
	Timestamp             string `json:"timestamp"`
	Random                string `json:"prev_randao" hex:"true"`
	SuggestedFeeRecipient string `json:"suggested_fee_recipient" hex:"true"`
}

type PayloadAttributesV2 struct {
	Timestamp             string               `json:"timestamp"`
	Random                string               `json:"prev_randao" hex:"true"`
	SuggestedFeeRecipient string               `json:"suggested_fee_recipient" hex:"true"`
	Withdrawals           []*shared.Withdrawal `json:"withdrawals"`
}

type PayloadAttributesV3 struct {
	Timestamp             string               `json:"timestamp"`
	Random                string               `json:"prev_randao" hex:"true"`
	SuggestedFeeRecipient string               `json:"suggested_fee_recipient" hex:"true"`
	Withdrawals           []*shared.Withdrawal `json:"withdrawals"`
	ParentBeaconBlockRoot string               `json:"parent_beacon_block_root" hex:"true"`
}

type BlobSidecarEvent struct {
	BlockRoot     string `json:"block_root"`
	Index         string `json:"index"`
	Slot          string `json:"slot"`
	KzgCommitment string `json:"kzg_commitment"`
	VersionedHash string `json:"versioned_hash"`
}

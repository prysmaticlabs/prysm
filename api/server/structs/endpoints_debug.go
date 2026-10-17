package structs

import (
	"encoding/json"
)

type GetBeaconStateV2Response struct {
	Version             string          `json:"version"`
	ExecutionOptimistic bool            `json:"execution_optimistic"`
	Finalized           bool            `json:"finalized"`
	Data                json.RawMessage `json:"data"` // represents the state values based on the version
}

type GetForkChoiceHeadsV2Response struct {
	Data []*ForkChoiceHead `json:"data"`
}

type ForkChoiceHead struct {
	Root                string `json:"root"`
	Slot                string `json:"slot"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type GetForkChoiceDumpResponse struct {
	JustifiedCheckpoint *Checkpoint              `json:"justified_checkpoint"`
	FinalizedCheckpoint *Checkpoint              `json:"finalized_checkpoint"`
	ForkChoiceNodes     []*ForkChoiceNode        `json:"fork_choice_nodes"`
	ExtraData           *ForkChoiceDumpExtraData `json:"extra_data"`
}

type ForkChoiceDumpExtraData struct {
	UnrealizedJustifiedCheckpoint *Checkpoint `json:"unrealized_justified_checkpoint"`
	UnrealizedFinalizedCheckpoint *Checkpoint `json:"unrealized_finalized_checkpoint"`
	ProposerBoostRoot             string      `json:"proposer_boost_root"`
	PreviousProposerBoostRoot     string      `json:"previous_proposer_boost_root"`
	HeadRoot                      string      `json:"head_root"`
}

type ForkChoiceNode struct {
	Slot               string                   `json:"slot"`
	BlockRoot          string                   `json:"block_root"`
	ParentRoot         string                   `json:"parent_root"`
	JustifiedEpoch     string                   `json:"justified_epoch"`
	FinalizedEpoch     string                   `json:"finalized_epoch"`
	Weight             string                   `json:"weight"`
	Validity           string                   `json:"validity"`
	ExecutionBlockHash string                   `json:"execution_block_hash"`
	ExtraData          *ForkChoiceNodeExtraData `json:"extra_data"`
}

type ForkChoiceNodeExtraData struct {
	UnrealizedJustifiedEpoch string `json:"unrealized_justified_epoch"`
	UnrealizedFinalizedEpoch string `json:"unrealized_finalized_epoch"`
	Balance                  string `json:"balance"`
	ExecutionOptimistic      bool   `json:"execution_optimistic"`
	TimeStamp                string `json:"timestamp"`
	Target                   string `json:"target"`
}

// DecoupledForkChoiceDumpResponse is the fork choice dump after the fork.
type DecoupledForkChoiceDumpResponse struct {
	JustifiedCheckpoint *Checkpoint                       `json:"justified_checkpoint"`
	FinalizedCheckpoint *Checkpoint                       `json:"finalized_checkpoint"`
	ForkChoiceNodes     []*DecoupledForkChoiceNode        `json:"fork_choice_nodes"`
	ExtraData           *DecoupledForkChoiceDumpExtraData `json:"extra_data"`
}

// DecoupledForkChoiceNode is a block in the dump.
type DecoupledForkChoiceNode struct {
	Slot               string                            `json:"slot"`
	BlockRoot          string                            `json:"block_root"`
	ParentRoot         string                            `json:"parent_root"`
	JustifiedEpoch     string                            `json:"justified_epoch"`
	FinalizedEpoch     string                            `json:"finalized_epoch"`
	Weight             string                            `json:"weight"`
	Validity           string                            `json:"validity"`
	ExecutionBlockHash string                            `json:"execution_block_hash"`
	ExtraData          *DecoupledForkChoiceNodeExtraData `json:"extra_data"`
}

// DecoupledForkChoiceNodeExtraData is a node's heights, counts and flags.
type DecoupledForkChoiceNodeExtraData struct {
	TargetHeight    string `json:"target_height"`
	TargetRoot      string `json:"target_root"`
	JustifiedHeight string `json:"justified_height"`
	JustifiedRoot   string `json:"justified_root"`
	FinalizedHeight string `json:"finalized_height"`
	FinalizedRoot   string `json:"finalized_root"`
	Votes           string `json:"votes"`
	SupportVotes    string `json:"support_votes"`
	Viable          bool   `json:"viable"`
	Confirmed       bool   `json:"confirmed"`
	Locked          bool   `json:"locked"`
	Processed       string `json:"processed"`
}

// DecoupledForkChoiceDumpExtraData is the store's head, confirmation, finality and locks.
type DecoupledForkChoiceDumpExtraData struct {
	Decoupled           bool                         `json:"decoupled"`
	Joined              bool                         `json:"joined"`
	Slot                string                       `json:"slot"`
	RestartTime         string                       `json:"restart_time"`
	HeadRoot            string                       `json:"head_root"`
	ConfirmedRoot       string                       `json:"confirmed_root"`
	LatestConfirmedRoot string                       `json:"latest_confirmed_root"`
	JustifiedHeight     string                       `json:"justified_height"`
	JustifiedSlot       string                       `json:"justified_slot"`
	FinalizedHeight     string                       `json:"finalized_height"`
	FinalizedSlot       string                       `json:"finalized_slot"`
	MaxHeight           string                       `json:"max_height"`
	FrameRoot           string                       `json:"frame_root"`
	Floor               string                       `json:"floor"`
	CommitteeSize       string                       `json:"committee_size"`
	Locks               []*DecoupledForkChoiceLock   `json:"locks"`
	FastConfirmed       []string                     `json:"fast_confirmed"`
	Saved               []*DecoupledForkChoiceSaved  `json:"saved"`
	Voters              []*DecoupledForkChoiceVoters `json:"voters"`
}

// DecoupledForkChoiceLock is a lock.
type DecoupledForkChoiceLock struct {
	Root     string `json:"root"`
	Slot     string `json:"slot"`
	Deadline string `json:"deadline"`
}

// DecoupledForkChoiceSaved is a slot's saved confirmation.
type DecoupledForkChoiceSaved struct {
	Slot string `json:"slot"`
	Root string `json:"root"`
}

// DecoupledForkChoiceVoters is a slot's voter count.
type DecoupledForkChoiceVoters struct {
	Slot  string `json:"slot"`
	Count string `json:"count"`
}

// GetDecoupledBlockSummaryResponse wraps a block summary.
type GetDecoupledBlockSummaryResponse struct {
	Data *DecoupledBlockSummary `json:"data"`
}

// DecoupledBlockSummary is a block's post-state FG values. J and F are the block's claim.
type DecoupledBlockSummary struct {
	Slot                string                             `json:"slot"`
	Root                string                             `json:"root"`
	Height              string                             `json:"height"`
	Entry               *CheckpointDecoupled               `json:"entry"`
	EntryDeferred       bool                               `json:"entry_deferred"`
	TargetParticipation string                             `json:"target_participation"`
	Progress            string                             `json:"progress"`
	ValidatorCount      string                             `json:"validator_count"`
	JustifiedHeight     string                             `json:"justified_height"`
	Justified           *CheckpointDecoupled               `json:"justified"`
	FinalizedHeight     string                             `json:"finalized_height"`
	Finalized           *CheckpointDecoupled               `json:"finalized"`
	PendingFinality     []*DecoupledPendingFinalitySummary `json:"pending_finality"`
}

// DecoupledPendingFinalitySummary is a pending finality pair.
type DecoupledPendingFinalitySummary struct {
	Height        string               `json:"height"`
	Target        *CheckpointDecoupled `json:"target"`
	Participation string               `json:"participation"`
}

type GetForkChoiceDumpV2Response struct {
	JustifiedCheckpoint *Checkpoint              `json:"justified_checkpoint"`
	FinalizedCheckpoint *Checkpoint              `json:"finalized_checkpoint"`
	ForkChoiceNodes     []*ForkChoiceNodeV2      `json:"fork_choice_nodes"`
	ExtraData           *ForkChoiceDumpExtraData `json:"extra_data"`
}

type ForkChoiceNodeV2 struct {
	PayloadStatus      string                     `json:"payload_status"`
	Slot               string                     `json:"slot"`
	BlockRoot          string                     `json:"block_root"`
	ParentRoot         string                     `json:"parent_root"`
	Weight             string                     `json:"weight"`
	Validity           string                     `json:"validity"`
	ExecutionBlockHash string                     `json:"execution_block_hash"`
	ExtraData          *ForkChoiceNodeV2ExtraData `json:"extra_data"`
}

type ForkChoiceNodeV2ExtraData struct {
	Balance             string `json:"balance"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
	TimeStamp           string `json:"timestamp"`

	Target                          string `json:"target,omitempty"`
	JustifiedEpoch                  string `json:"justified_epoch,omitempty"`
	FinalizedEpoch                  string `json:"finalized_epoch,omitempty"`
	UnrealizedJustifiedEpoch        string `json:"unrealized_justified_epoch,omitempty"`
	UnrealizedFinalizedEpoch        string `json:"unrealized_finalized_epoch,omitempty"`
	PayloadAttesterCount            string `json:"payload_attester_count,omitempty"`
	PayloadAvailabilityYesCount     string `json:"payload_availability_yes_count,omitempty"`
	PayloadDataAvailabilityYesCount string `json:"payload_data_availability_yes_count,omitempty"`

	GasLimit string `json:"gas_limit,omitempty"`
}

type GetDebugDataColumnSidecarsResponse struct {
	Version             string          `json:"version"`
	ExecutionOptimistic bool            `json:"execution_optimistic"`
	Finalized           bool            `json:"finalized"`
	Data                json.RawMessage `json:"data"` // []*DataColumnSidecar pre-Gloas, []*DataColumnSidecarGloas post-Gloas
}

type DataColumnSidecar struct {
	Index                        string                   `json:"index"`
	Column                       []string                 `json:"column"`
	KzgCommitments               []string                 `json:"kzg_commitments"`
	KzgProofs                    []string                 `json:"kzg_proofs"`
	SignedBeaconBlockHeader      *SignedBeaconBlockHeader `json:"signed_block_header"`
	KzgCommitmentsInclusionProof []string                 `json:"kzg_commitments_inclusion_proof"`
}

type DataColumnSidecarGloas struct {
	Index           string   `json:"index"`
	Column          []string `json:"column"`
	KzgProofs       []string `json:"kzg_proofs"`
	Slot            string   `json:"slot"`
	BeaconBlockRoot string   `json:"beacon_block_root"`
}

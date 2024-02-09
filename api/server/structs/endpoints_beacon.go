package structs

import (
	"encoding/json"
)

type BlockRootResponse struct {
	Data                *BlockRoot `json:"data"`
	ExecutionOptimistic bool       `json:"execution_optimistic"`
	Finalized           bool       `json:"finalized"`
}

type BlockRoot struct {
	Root string `json:"root"`
}

type GetCommitteesResponse struct {
	Data                []*Committee `json:"data"`
	ExecutionOptimistic bool         `json:"execution_optimistic"`
	Finalized           bool         `json:"finalized"`
}

type ListAttestationsResponse struct {
	Data []*Attestation `json:"data"`
}

type SubmitAttestationsRequest struct {
	Data []*Attestation `json:"data"`
}

type ListVoluntaryExitsResponse struct {
	Data []*SignedVoluntaryExit `json:"data"`
}

type SubmitSyncCommitteeSignaturesRequest struct {
	Data []*SyncCommitteeMessage `json:"data"`
}

type GetStateForkResponse struct {
	Data                *Fork `json:"data"`
	ExecutionOptimistic bool  `json:"execution_optimistic"`
	Finalized           bool  `json:"finalized"`
}

type GetFinalityCheckpointsResponse struct {
	ExecutionOptimistic bool                 `json:"execution_optimistic"`
	Finalized           bool                 `json:"finalized"`
	Data                *FinalityCheckpoints `json:"data"`
}

type FinalityCheckpoints struct {
	PreviousJustified *Checkpoint `json:"previous_justified"`
	CurrentJustified  *Checkpoint `json:"current_justified"`
	Finalized         *Checkpoint `json:"finalized"`
}

type GetGenesisResponse struct {
	Data *Genesis `json:"data"`
}

type Genesis struct {
	GenesisTime           string `json:"genesis_time"`
	GenesisValidatorsRoot string `json:"genesis_validators_root"`
	GenesisForkVersion    string `json:"genesis_fork_version"`
}

type GetBlockHeadersResponse struct {
	Data                []*SignedBeaconBlockHeaderContainer `json:"data"`
	ExecutionOptimistic bool                                `json:"execution_optimistic"`
	Finalized           bool                                `json:"finalized"`
}

type GetBlockHeaderResponse struct {
	ExecutionOptimistic bool                              `json:"execution_optimistic"`
	Finalized           bool                              `json:"finalized"`
	Data                *SignedBeaconBlockHeaderContainer `json:"data"`
}

type GetValidatorsRequest struct {
	Ids      []string `json:"ids"`
	Statuses []string `json:"statuses"`
}

type GetValidatorsResponse struct {
	ExecutionOptimistic bool                  `json:"execution_optimistic"`
	Finalized           bool                  `json:"finalized"`
	Data                []*ValidatorContainer `json:"data"`
}

type GetValidatorResponse struct {
	ExecutionOptimistic bool                `json:"execution_optimistic"`
	Finalized           bool                `json:"finalized"`
	Data                *ValidatorContainer `json:"data"`
}

type GetValidatorBalancesResponse struct {
	ExecutionOptimistic bool                `json:"execution_optimistic"`
	Finalized           bool                `json:"finalized"`
	Data                []*ValidatorBalance `json:"data"`
}

type ValidatorContainer struct {
	Index     string     `json:"index"`
	Balance   string     `json:"balance"`
	Status    string     `json:"status"`
	Validator *Validator `json:"validator"`
}

type ValidatorBalance struct {
	Index   string `json:"index"`
	Balance string `json:"balance"`
}

type GetBlockResponse struct {
	Data *SignedBlock `json:"data"`
}

type GetBlockV2Response struct {
	Version             string       `json:"version"`
	ExecutionOptimistic bool         `json:"execution_optimistic"`
	Finalized           bool         `json:"finalized"`
	Data                *SignedBlock `json:"data"`
}

type SignedBlock struct {
	Message   json.RawMessage `json:"message"` // represents the block values based on the version
	Signature string          `json:"signature"`
}

type GetBlockAttestationsResponse struct {
	ExecutionOptimistic bool           `json:"execution_optimistic"`
	Finalized           bool           `json:"finalized"`
	Data                []*Attestation `json:"data"`
}

type GetStateRootResponse struct {
	ExecutionOptimistic bool       `json:"execution_optimistic"`
	Finalized           bool       `json:"finalized"`
	Data                *StateRoot `json:"data"`
}

type StateRoot struct {
	Root string `json:"root"`
}

type GetRandaoResponse struct {
	ExecutionOptimistic bool    `json:"execution_optimistic"`
	Finalized           bool    `json:"finalized"`
	Data                *Randao `json:"data"`
}

type Randao struct {
	Randao string `json:"randao"`
}

type GetSyncCommitteeResponse struct {
	ExecutionOptimistic bool                     `json:"execution_optimistic"`
	Finalized           bool                     `json:"finalized"`
	Data                *SyncCommitteeValidators `json:"data"`
}

type SyncCommitteeValidators struct {
	Validators          []string   `json:"validators"`
	ValidatorAggregates [][]string `json:"validator_aggregates"`
}

type BLSToExecutionChangesPoolResponse struct {
	Data []*SignedBLSToExecutionChange `json:"data"`
}

type GetAttesterSlashingsResponse struct {
	Data []*AttesterSlashing `json:"data"`
}

type GetProposerSlashingsResponse struct {
	Data []*ProposerSlashing `json:"data"`
}

type GetWeakSubjectivityResponse struct {
	Data *WeakSubjectivityData `json:"data"`
}

type WeakSubjectivityData struct {
	WsCheckpoint *Checkpoint `json:"ws_checkpoint"`
	StateRoot    string      `json:"state_root"`
}

type GetDepositSnapshotResponse struct {
	Data *DepositSnapshot `json:"data"`
}

type DepositSnapshot struct {
	Finalized            []string `json:"finalized"`
	DepositRoot          string   `json:"deposit_root"`
	DepositCount         string   `json:"deposit_count"`
	ExecutionBlockHash   string   `json:"execution_block_hash"`
	ExecutionBlockHeight string   `json:"execution_block_height"`
}

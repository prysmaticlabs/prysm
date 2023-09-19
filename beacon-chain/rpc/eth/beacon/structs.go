package beacon

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
)

type BlockRootResponse struct {
	Data *struct {
		Root string `json:"root"`
	} `json:"data"`
	ExecutionOptimistic bool `json:"execution_optimistic"`
	Finalized           bool `json:"finalized"`
}

type GetCommitteesResponse struct {
	Data                []*shared.Committee `json:"data"`
	ExecutionOptimistic bool                `json:"execution_optimistic"`
	Finalized           bool                `json:"finalized"`
}

type DepositContractResponse struct {
	Data *struct {
		ChainId uint64 `json:"chain_id"`
		Address string `json:"address"`
	} `json:"data"`
}

type ListAttestationsResponse struct {
	Data []*shared.Attestation `json:"data"`
}

type SubmitAttestationsRequest struct {
	Data []*shared.Attestation `json:"data"`
}

type ListVoluntaryExitsResponse struct {
	Data []*shared.SignedVoluntaryExit `json:"data"`
}

type SubmitSyncCommitteeSignaturesRequest struct {
	Data []*shared.SyncCommitteeMessage `json:"data"`
}

type GetStateForkResponse struct {
	Data                *shared.Fork `json:"data"`
	ExecutionOptimistic bool         `json:"execution_optimistic"`
	Finalized           bool         `json:"finalized"`
}

type GetFinalityCheckpointsResponse struct {
	ExecutionOptimistic bool                 `json:"execution_optimistic"`
	Finalized           bool                 `json:"finalized"`
	Data                *FinalityCheckpoints `json:"data"`
}

type FinalityCheckpoints struct {
	PreviousJustified *shared.Checkpoint `json:"previous_justified"`
	CurrentJustified  *shared.Checkpoint `json:"current_justified"`
	Finalized         *shared.Checkpoint `json:"finalized"`
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
	Data                []*shared.SignedBeaconBlockHeaderContainer `json:"data"`
	ExecutionOptimistic bool                                       `json:"execution_optimistic"`
	Finalized           bool                                       `json:"finalized"`
}

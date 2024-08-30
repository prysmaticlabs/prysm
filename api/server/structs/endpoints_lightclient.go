package structs

// the interface is used in other structs to reference a light client header object regardless of its version
type lightClientHeader interface {
	isLightClientHeader()
}

type LightClientHeader struct {
	Beacon *BeaconBlockHeader `json:"beacon"`
}

func (a *LightClientHeader) isLightClientHeader() {}

type LightClientHeaderCapella struct {
	Beacon          *BeaconBlockHeader             `json:"beacon"`
	Execution       *ExecutionPayloadHeaderCapella `json:"execution"`
	ExecutionBranch []string                       `json:"execution_branch"`
}

func (a *LightClientHeaderCapella) isLightClientHeader() {}

type LightClientHeaderDeneb struct {
	Beacon          *BeaconBlockHeader           `json:"beacon"`
	Execution       *ExecutionPayloadHeaderDeneb `json:"execution"`
	ExecutionBranch []string                     `json:"execution_branch"`
}

func (a *LightClientHeaderDeneb) isLightClientHeader() {}

type LightClientBootstrap struct {
	Header                     lightClientHeader `json:"header"`
	CurrentSyncCommittee       *SyncCommittee    `json:"current_sync_committee"`
	CurrentSyncCommitteeBranch []string          `json:"current_sync_committee_branch"`
}

type LightClientBootstrapResponse struct {
	Version string                `json:"version"`
	Data    *LightClientBootstrap `json:"data"`
}

type LightClientUpdate struct {
	AttestedHeader          lightClientHeader `json:"attested_header"`
	NextSyncCommittee       *SyncCommittee    `json:"next_sync_committee,omitempty"`
	FinalizedHeader         lightClientHeader `json:"finalized_header,omitempty"`
	SyncAggregate           *SyncAggregate    `json:"sync_aggregate"`
	NextSyncCommitteeBranch []string          `json:"next_sync_committee_branch,omitempty"`
	FinalityBranch          []string          `json:"finality_branch,omitempty"`
	SignatureSlot           string            `json:"signature_slot"`
}

type LightClientUpdateWithVersion struct {
	Version string             `json:"version"`
	Data    *LightClientUpdate `json:"data"`
}

type LightClientUpdatesByRangeResponse struct {
	Updates []*LightClientUpdateWithVersion `json:"updates"`
}

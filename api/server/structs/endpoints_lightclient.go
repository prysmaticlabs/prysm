package structs

type LightClientHeader struct {
	Beacon *BeaconBlockHeader `json:"beacon"`
}

type LightClientBootstrapResponse struct {
	Version string                `json:"version"`
	Data    *LightClientBootstrap `json:"data"`
}

type LightClientBootstrap struct {
	Header                     *LightClientHeader `json:"header"`
	CurrentSyncCommittee       *SyncCommittee     `json:"current_sync_committee"`
	CurrentSyncCommitteeBranch []string           `json:"current_sync_committee_branch"`
}

type LightClientUpdate struct {
	AttestedHeader          *LightClientHeader `json:"attested_header"`
	NextSyncCommittee       *SyncCommittee     `json:"next_sync_committee,omitempty"`
	FinalizedHeader         *LightClientHeader `json:"finalized_header,omitempty"`
	SyncAggregate           *SyncAggregate     `json:"sync_aggregate"`
	NextSyncCommitteeBranch []string           `json:"next_sync_committee_branch,omitempty"`
	FinalityBranch          []string           `json:"finality_branch,omitempty"`
	SignatureSlot           string             `json:"signature_slot"`
}

type LightClientUpdateWithVersion struct {
	Version string             `json:"version"`
	Data    *LightClientUpdate `json:"data"`
}

type LightClientUpdatesByRangeResponse struct {
	Updates []*LightClientUpdateWithVersion `json:"updates"`
}

package lightclient

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
)

type LightClientBootstrapResponse struct {
	Version string                `json:"version"`
	Data    *LightClientBootstrap `json:"data"`
}

type LightClientBootstrap struct {
	Header                     *apimiddleware.BeaconBlockHeaderJson `json:"header"`
	CurrentSyncCommittee       *apimiddleware.SyncCommitteeJson     `json:"current_sync_committee"`
	CurrentSyncCommitteeBranch []string                             `json:"current_sync_committee_branch"`
}

type LightClientUpdate struct {
	AttestedHeader          *apimiddleware.BeaconBlockHeaderJson `json:"attested_header"`
	NextSyncCommittee       *apimiddleware.SyncCommitteeJson     `json:"next_sync_committee"`
	FinalizedHeader         *apimiddleware.BeaconBlockHeaderJson `json:"finalized_header"`
	SyncAggregate           *apimiddleware.SyncAggregateJson     `json:"sync_aggregate"`
	NextSyncCommitteeBranch []string                             `json:"next_sync_committee_branch"`
	FinalityBranch          []string                             `json:"finality_branch"`
	SignatureSlot           string                               `json:"signature_slot"`
}

type LightClientUpdateWithVersion struct {
	Version string             `json:"version"`
	Data    *LightClientUpdate `json:"data"`
}

type LightClientUpdatesByRangeResponse struct {
	Updates []*LightClientUpdateWithVersion `json:"updates"`
}

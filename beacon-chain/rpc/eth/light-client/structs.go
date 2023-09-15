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

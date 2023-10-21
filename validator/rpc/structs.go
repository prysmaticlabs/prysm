package rpc

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager"
)

type SetVoluntaryExitResponse struct {
	Data *shared.SignedVoluntaryExit `json:"data"`
}

// remote keymanager api
type ListRemoteKeysResponse struct {
	Data []*RemoteKey `json:"data"`
}

type RemoteKey struct {
	Pubkey   string `json:"pubkey"`
	Url      string `json:"url"`
	Readonly bool   `json:"readonly"`
}

type ImportRemoteKeysRequest struct {
	RemoteKeys []*RemoteKey `json:"remote_keys"`
}

type DeleteRemoteKeysRequest struct {
	Pubkeys []string `json:"pubkeys"`
}

type RemoteKeysResponse struct {
	Data []*keymanager.KeyStatus `json:"data"`
}

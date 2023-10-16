package rpc

import "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"

type SetVoluntaryExitResponse struct {
	Data *shared.SignedVoluntaryExit `json:"data"`
}

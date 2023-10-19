package rpc

import "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"

type SetVoluntaryExitResponse struct {
	Data *shared.SignedVoluntaryExit `json:"data"`
}

type GasLimitMetaData struct {
	Pubkey   string `json:"pubkey"`
	GasLimit string `json:"gas_limit"`
}

type GetGasLimitResponse struct {
	Data *GasLimitMetaData `json:"data"`
}

type SetGasLimitRequest struct {
	GasLimit string `json:"gas_limit"`
}

type DeleteGasLimitRequest struct {
	Pubkey string `json:"pubkey"`
}

package validator

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/lookup"
)

type Server struct {
	Stater      lookup.Stater
	CoreService *core.Service
}

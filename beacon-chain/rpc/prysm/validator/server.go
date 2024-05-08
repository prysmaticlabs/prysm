package validator

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
)

type Server struct {
	CoreService *core.Service
}

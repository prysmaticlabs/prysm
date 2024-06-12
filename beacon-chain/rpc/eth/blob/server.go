package blob

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/lookup"
)

type Server struct {
	Blocker lookup.Blocker
}

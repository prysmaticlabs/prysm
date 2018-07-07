package p2p

import (
	"github.com/prysmaticlabs/geth-sharding/sharding"
)

// Ensure that server implements service.
var _ = sharding.Service(&Server{})

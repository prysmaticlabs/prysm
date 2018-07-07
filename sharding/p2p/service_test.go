package p2p

import "github.com/prysmaticlabs/geth-sharding/sharding/types"

// Ensure that server implements service.
var _ = types.Service(&Server{})

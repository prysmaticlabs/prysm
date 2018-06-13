package p2p

import "github.com/ethereum/go-ethereum/sharding"

// Ensure that server implements service.
var _ = sharding.Service(&Server{})

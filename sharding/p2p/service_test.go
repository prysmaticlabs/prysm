package p2p

import (
	"github.com/ethereum/go-ethereum/sharding"
)

// Verifies that Server implements the ShardP2P interface.
var _ = sharding.ShardP2P(&Server{})

package txpool

import "github.com/ethereum/go-ethereum/sharding"

// Verifies that ShardTXPool implements the TXPool interface.
var _ = sharding.TXPool(&ShardTXPool{})

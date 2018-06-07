package node

import "github.com/ethereum/go-ethereum/sharding"

// Verifies that ShardEthereum implements the Node interface.
var _ = sharding.Node(&ShardEthereum{})

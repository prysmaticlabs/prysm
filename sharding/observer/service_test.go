package observer

import "github.com/ethereum/go-ethereum/sharding"

// Verifies that Observer implements the Actor interface.
var _ = sharding.Actor(&Observer{})

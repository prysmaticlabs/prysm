package proposer

import "github.com/ethereum/go-ethereum/sharding"

// Verifies that Proposer implements the Actor interface.
var _ = sharding.Actor(&Proposer{})

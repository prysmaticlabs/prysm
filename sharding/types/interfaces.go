package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// Node defines a a sharding-enabled Ethereum instance that provides
// full control and shared access of necessary components and services
// for a sharded Ethereum blockchain.
type Node interface {
	Start()
	Close()
}

// Actor refers to either an attester, proposer, or observer in the sharding spec.
type Actor interface {
	Service
	// TODO: will actors have actor-specific methods? To be decided.
}

// CollationFetcher defines functionality for a struct that is able to extract
// respond with collation information to the caller. Shard implements this interface.
type CollationFetcher interface {
	CollationByHeaderHash(headerHash *common.Hash) (*Collation, error)
}

// Service is an individual protocol that can be registered into a node. Having a sharding
// node maintain a service registry allows for easy, shared-dependencies. For example,
// a proposer service might depend on a p2p server, a txpool, an smc client, etc.
type Service interface {
	// Start is called after all services have been constructed to
	// spawn any goroutines required by the service.
	Start()
	// Stop terminates all goroutines belonging to the service,
	// blocking until they are all terminated.
	Stop() error
}

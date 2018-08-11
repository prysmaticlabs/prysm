package types

import (
	"github.com/ethereum/go-ethereum/common"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared"
)

// Node defines a a sharding-enabled Ethereum instance that provides
// full control and shared access of necessary components and services
// for a sharded Ethereum blockchain.
type Node interface {
	Start()
	Close()
}

// Actor refers to either a attester, proposer in the sharding spec.
type Actor interface {
	shared.Service
	// TODO: will actors have actor-specific methods? To be decided.
}

// CollationFetcher defines functionality for a struct that is able to extract
// respond with collation information to the caller. Shard implements this interface.
type CollationFetcher interface {
	CollationByHeaderHash(headerHash *common.Hash) (*Collation, error)
}

// RPCClient defines a struct that opens up RPC client services via gRPC.
type RPCClient interface {
	BeaconServiceClient() pb.BeaconServiceClient
}

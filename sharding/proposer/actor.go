package proposer

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	node sharding.ShardingClient
}

// NewProposer creates a struct instance. It is initialized and
// registered as a service upon start of a sharding node.
// Has access to the public methods of this node.
func NewProposer(node sharding.ShardingClient) (*Proposer, error) {
	// Initializes a  directory persistent db.
	return &Proposer{node}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() error {
	log.Info("Starting proposer service")
	// TODO: Propose collations.
	return nil
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Info("Stopping proposer service")
	return nil
}

package proposer

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/node"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	node    node.Node
	shardDB database.ShardBackend
}

// NewProposer creates a struct instance. It is initialized and
// registered as a service upon start of a sharding node.
// Has access to the public methods of this node.
func NewProposer(node node.Node) (*Proposer, error) {
	// Initializes a shardchaindata directory persistent db.
	shardDB, err := database.NewShardDB(node.DataDirFlag(), "shardchaindata")
	if err != nil {
		return nil, err
	}
	return &Proposer{node, shardDB}, nil
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

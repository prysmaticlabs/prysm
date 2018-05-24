package notary

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/node"
)

// Notary holds functionality required to run a collation notary
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Notary struct {
	node    node.Node
	shardDB database.ShardBackend
}

// NewNotary creates a new notary instance.
func NewNotary(node node.Node) (*Notary, error) {
	// Initializes a shardDB that writes to disk at /path/to/datadir/shardchaindata.
	// This DB can be used by the Notary service to create Shard struct
	// instances.
	shardDB, err := database.NewShardDB(node.DataDirFlag(), "shardchaindata")
	if err != nil {
		return nil, err
	}
	// return &Notary{node, shardDB}, nil
	return &Notary{node: node}, nil
}

// Start the main routine for a notary.
func (n *Notary) Start() error {
	log.Info("Starting notary service")

	// TODO: handle this better through goroutines. Right now, these methods
	// have their own nested channels and goroutines within them. We need
	// to make this as flat as possible at the Notary layer.
	if n.node.DepositFlagSet() {
		if err := joinNotaryPool(n.node); err != nil {
			return err
		}
	}

	return subscribeBlockHeaders(n.node)
}

// Stop the main loop for notarizing collations.
func (n *Notary) Stop() error {
	log.Info("Stopping notary service")
	return nil
}

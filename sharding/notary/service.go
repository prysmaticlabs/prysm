package notary

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/node"
)

// Notary holds functionality required to run a collation notary
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Notary struct {
	node node.Node
}

// NewNotary creates a new notary instance.
func NewNotary(node node.Node) (*Notary, error) {
	return &Notary{node}, nil
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

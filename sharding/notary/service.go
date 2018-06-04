package notary

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
)

// Notary holds functionality required to run a collation notary
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Notary struct {
	smcClient sharding.SMCClient
	shardp2p  sharding.ShardP2P
}

// NewNotary creates a new notary instance.
func NewNotary(smcClient sharding.SMCClient, shardp2p sharding.ShardP2P) (*Notary, error) {
	return &Notary{smcClient, shardp2p}, nil
}

// Start the main routine for a notary.
func (n *Notary) Start() error {
	log.Info("Starting notary service")

	// TODO: handle this better through goroutines. Right now, these methods
	// have their own nested channels and goroutines within them. We need
	// to make this as flat as possible at the Notary layer.
	if n.smcClient.DepositFlag() {
		if err := joinNotaryPool(n.smcClient); err != nil {
			return err
		}
	}

	return subscribeBlockHeaders(n.smcClient)
}

// Stop the main loop for notarizing collations.
func (n *Notary) Stop() error {
	log.Info("Stopping notary service")
	return nil
}

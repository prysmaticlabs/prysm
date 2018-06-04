// Package observer launches a service attached to the sharding node
// that simply observes activity across the sharded Ethereum network.
package observer

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
)

// Observer holds functionality required to run an observer service
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Observer struct {
	node sharding.Node
}

// NewObserver creates a new observer instance.
func NewObserver(node sharding.Node) (*Observer, error) {
	return &Observer{node}, nil
}

// Start the main routine for an observer.
func (o *Observer) Start() error {
	log.Info("Starting shard observer service")
	return nil
}

// Stop the main loop for observing the shard network.
func (o *Observer) Stop() error {
	log.Info("Stopping shard observer service")
	return nil
}

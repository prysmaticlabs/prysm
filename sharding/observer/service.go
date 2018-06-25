// Package observer launches a service attached to the sharding node
// that simply observes activity across the sharded Ethereum network.
package observer

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/p2p"
)

// Observer holds functionality required to run an observer service
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Observer struct {
	p2p       *p2p.Server
	dbService *database.ShardDB
	shardID   int
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewObserver creates a struct instance of a observer service,
// it will have access to a p2p server and a shardChainDb.
func NewObserver(p2p *p2p.Server, dbService *database.ShardDB, shardID int) (*Observer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Observer{p2p, dbService, shardID, ctx, cancel}, nil
}

// Start the main loop for observer service.
func (o *Observer) Start() {
	log.Info(fmt.Sprintf("Starting observer service"))
	// shard := sharding.NewShard(big.NewInt(int64(o.shardID)), o.dbService.DB())
}

// Stop the main loop for observer service.
func (o *Observer) Stop() error {
	// Triggers a cancel call in the service's context which shuts down every goroutine
	// in this service.
	defer o.cancel()
	log.Info(fmt.Sprintf("Stopping observer service"))
	return nil
}

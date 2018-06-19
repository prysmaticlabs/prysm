// Package observer launches a service attached to the sharding node
// that simply observes activity across the sharded Ethereum network.
package observer

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/p2p"
)

// Observer holds functionality required to run an observer service
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Observer struct {
	p2p          *p2p.Server
	shardChainDb ethdb.Database
	shardID      int
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewObserver creates a new observer instance.
func NewObserver(p2p *p2p.Server, shardChainDb ethdb.Database, shardID int) (*Observer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	return &Observer{p2p, shardChainDb, shardID, ctx, cancel}, nil
}

// Start the main routine for an observer.
func (o *Observer) Start() {
	log.Info(fmt.Sprintf("Starting observer service in shard %d", o.shardID))

	//	go func() {
	//		ticker := time.NewTicker(6 * time.Second)
	//		defer ticker.Stop()
	//
	//		for {
	//			select {
	//			case <-ticker.C:
	//				o.p2p.Broadcast(nil)
	//			case <-o.ctx.Done():
	//				return
	//			}
	//		}
	//	}()
}

// Stop the main loop for observing the shard network.
func (o *Observer) Stop() error {
	log.Info(fmt.Sprintf("Stopping observer service in shard %d", o.shardID))
	o.cancel()
	return nil
}

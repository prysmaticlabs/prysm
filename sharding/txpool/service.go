// Package txpool handles incoming transactions for a sharded Ethereum blockchain.
package txpool

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/p2p"
)

// ShardTXPool handles a transaction pool for a sharded system.
type ShardTXPool struct {
	shardp2p *p2p.Server
}

// NewShardTXPool creates a new observer instance.
func NewShardTXPool(shardp2p *p2p.Server) (*ShardTXPool, error) {
	return &ShardTXPool{shardp2p}, nil
}

// Start the main routine for a shard transaction pool.
func (p *ShardTXPool) Start() {
	log.Info("Starting shard txpool service")
}

// Stop the main loop for a transaction pool in the shard network.
func (p *ShardTXPool) Stop() error {
	log.Info("Stopping shard txpool service")
	return nil
}

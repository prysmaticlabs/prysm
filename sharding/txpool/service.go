// Package txpool handles incoming transactions for a sharded Ethereum blockchain.
package txpool

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
)

// ShardTXPool handles a transaction pool for a sharded system.
type ShardTXPool struct {
	p2p              sharding.ShardP2P
	transactionsFeed *event.Feed
}

// NewShardTXPool creates a new observer instance.
func NewShardTXPool(p2p sharding.ShardP2P) (*ShardTXPool, error) {
	return &ShardTXPool{p2p: p2p, transactionsFeed: new(event.Feed)}, nil
}

// Start the main routine for a shard transaction pool.
func (pool *ShardTXPool) Start() error {
	log.Info("Starting shard txpool service")
	go pool.generateTestTransactions()
	return nil
}

// Stop the main loop for a transaction pool in the shard network.
func (pool *ShardTXPool) Stop() error {
	log.Info("Stopping shard txpool service")
	return nil
}

func (pool *ShardTXPool) TransactionsFeed() *event.Feed {
	return pool.transactionsFeed
}

func (pool *ShardTXPool) generateTestTransactions() {
	for {
		nsent := pool.transactionsFeed.Send(1)
		log.Info(fmt.Sprintf("Sent transaction to %d subscribers", nsent))
		time.Sleep(time.Second)
	}
}

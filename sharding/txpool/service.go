// Package txpool handles incoming transactions for a sharded Ethereum blockchain.
package txpool

import (
	"fmt"
	"time"

	"crypto/rand"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
)

// ShardTXPool handles a transaction pool for a sharded system.
type ShardTXPool struct {
	p2p              sharding.ShardP2P
	transactionsFeed *event.Feed
	ticker           *time.Ticker
}

// NewShardTXPool creates a new observer instance.
func NewShardTXPool(p2p sharding.ShardP2P) (*ShardTXPool, error) {
	return &ShardTXPool{p2p: p2p, transactionsFeed: new(event.Feed)}, nil
}

// Start the main routine for a shard transaction pool.
func (p *ShardTXPool) Start() {
	log.Info("Starting shard txpool service")
	go p.sendTestTransaction()
}

// Stop the main loop for a transaction pool in the shard network.
func (p *ShardTXPool) Stop() error {
	log.Info("Stopping shard txpool service")
	p.ticker.Stop()
	return nil
}

func (p *ShardTXPool) TransactionsFeed() *event.Feed {
	return p.transactionsFeed
}

func (p *ShardTXPool) sendTestTransaction() {
	p.ticker = time.NewTicker(5 * time.Second)

	for range p.ticker.C {
		tx := createTestTransaction()
		nsent := p.transactionsFeed.Send(tx)
		log.Info(fmt.Sprintf("Sent transaction %v to %d subscribers", tx, nsent))
	}
}

func createTestTransaction() *types.Transaction {
	data := make([]byte, 1024)
	rand.Read(data)
	return types.NewTransaction(0, common.HexToAddress("0x0"), nil, 0, nil, data)
}

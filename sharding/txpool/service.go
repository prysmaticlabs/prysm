// Package txpool handles incoming transactions for a sharded Ethereum blockchain.
package txpool

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
)

// TXPool handles a transaction pool for a sharded system.
type TXPool struct {
	p2p              *p2p.Server
	transactionsFeed *event.Feed
	ticker           *time.Ticker
}

// NewTXPool creates a new observer instance.
func NewTXPool(p2p *p2p.Server) (*TXPool, error) {
	return &TXPool{p2p: p2p, transactionsFeed: new(event.Feed)}, nil
}

// Start the main routine for a shard transaction pool.
func (p *TXPool) Start() {
	log.Info("Starting shard txpool service")
	go p.sendTestTransaction()
}

// Stop the main loop for a transaction pool in the shard network.
func (p *TXPool) Stop() error {
	log.Info("Stopping shard txpool service")
	p.ticker.Stop()
	return nil
}

func (p *TXPool) TransactionsFeed() *event.Feed {
	return p.transactionsFeed
}

// sendTestTransaction sends a transaction with random bytes over a 5 second interval.
// This method is for testing purposes only, and will be replaced by a more functional CLI tool.
func (p *TXPool) sendTestTransaction() {
	p.ticker = time.NewTicker(5 * time.Second)

	for range p.ticker.C {
		tx := createTestTransaction()
		nsent := p.transactionsFeed.Send(tx)
		log.Info(fmt.Sprintf("Sent transaction %x to %d subscribers", tx.Hash(), nsent))
	}
}

func createTestTransaction() *types.Transaction {
	data := make([]byte, 1024)
	rand.Read(data)
	return types.NewTransaction(0, common.HexToAddress("0x0"), nil, 0, nil, data)
}

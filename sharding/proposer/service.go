// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"math/big"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/event"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	client       *mainchain.SMCClient
	shardp2p     sharding.ShardP2P
	txpool       sharding.TXPool
	txpoolSub	 event.Subscription
	shardChainDb ethdb.Database
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a shardp2p network,
// and a shard transaction pool.
func NewProposer(client *mainchain.SMCClient, shardp2p sharding.ShardP2P, txpool sharding.TXPool, shardChainDb ethdb.Database) (*Proposer, error) {
	// Initializes a  directory persistent db.
	return &Proposer{client, shardp2p, txpool, nil, shardChainDb}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() error {
	log.Info("Starting proposer service")

	go p.subscribeTransactions()

	// TODO: Create and use CLI flag for shardID
	shardID := big.NewInt(0)

	// Get current block number.
	blockNumber, err := p.client.ChainReader().BlockByNumber(context.Background(), nil)
	if err != nil {
		return err
	}
	period := new(big.Int).Div(blockNumber.Number(), big.NewInt(sharding.PeriodLength))

	// Create collation.
	collation, err := createCollation(p.client, shardID, period, txs)
	if err != nil {
		return err
	}

	// Check SMC if we can submit header before addHeader
	canAdd, err := checkHeaderAdded(p.client, shardID, period)
	if err != nil {
		return err
	}
	if canAdd {
		addHeader(p.client, collation)
	}

	return nil
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Info("Stopping proposer service")
	p.txpoolSub.Unsubscribe()
	return nil
}

func (p *Proposer) subscribeTransactions() {
	requests := make(chan int)
	p.txpoolSub = p.txpool.TransactionsFeed().Subscribe(requests)

	for {
		timeout := time.NewTimer(10 * time.Second)
		select {
		case v := <-requests:
			log.Info(fmt.Sprintf("Received transaction id: %d", v))
		case <-timeout.C:
			log.Error("Subscriber timed out")
		case err := <-p.txpoolSub.Err():
			log.Error("Subscriber failed: %v", err)
			timeout.Stop()
			return
		}
		timeout.Stop()
	}
}

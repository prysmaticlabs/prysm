// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	client       *mainchain.SMCClient
	shardp2p     sharding.ShardP2P
	txpool       sharding.TXPool
	txpoolSub    event.Subscription
	shardChainDb ethdb.Database
	shardID      int
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a shardp2p network,
// and a shard transaction pool.
func NewProposer(client *mainchain.SMCClient, shardp2p sharding.ShardP2P, txpool sharding.TXPool, shardChainDb ethdb.Database, shardID int) (*Proposer, error) {
	// Initializes a  directory persistent db.
	return &Proposer{client, shardp2p, txpool, nil, shardChainDb, shardID}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() {
	log.Info(fmt.Sprintf("Starting proposer service in shard %d", p.shardID))
	go p.proposeCollations()
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Info(fmt.Sprintf("Stopping proposer service in shard %d", p.shardID))
	p.txpoolSub.Unsubscribe()
	return nil
}

func (p *Proposer) proposeCollations() {
	go p.subscribeTransactions()
}

func (p *Proposer) createCollation(txs []*types.Transaction) error {
	// Get current block number.
	blockNumber, err := p.client.ChainReader().BlockByNumber(context.Background(), nil)
	if err != nil {
		log.Error(fmt.Sprintf("Could not fetch current block number: %v", err))
		return err
	}
	period := new(big.Int).Div(blockNumber.Number(), big.NewInt(sharding.PeriodLength))

	// Create collation.
	collation, err := createCollation(p.client, big.NewInt(int64(p.shardID)), period, txs)
	if err != nil {
		log.Error(fmt.Sprintf("Could not create collation: %v", err))
		return err
	}

	// Check SMC if we can submit header before addHeader
	canAdd, err := checkHeaderAdded(p.client, big.NewInt(int64(p.shardID)), period)
	if err != nil {
		log.Error(fmt.Sprintf("Could not check if we can submit header: %v", err))
		return err
	}
	if canAdd {
		addHeader(p.client, collation)
	}

	return nil
}

func (p *Proposer) subscribeTransactions() {
	requests := make(chan *types.Transaction)
	p.txpoolSub = p.txpool.TransactionsFeed().Subscribe(requests)

	for {
		timeout := time.NewTimer(10 * time.Second)
		select {
		case tx := <-requests:
			if err := p.createCollation([]*types.Transaction{tx}); err == nil {
				log.Error("Create collation failed: %v", tx)
			}
			log.Info(fmt.Sprintf("Received transaction id: %v", tx))
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

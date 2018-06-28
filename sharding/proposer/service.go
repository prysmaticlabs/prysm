// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/p2p/messages"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/txpool"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	config       *params.Config
	client       *mainchain.SMCClient
	p2p          *p2p.Server
	txpool       *txpool.TXPool
	txpoolSub    event.Subscription
	shardChainDb ethdb.Database
	shard        *sharding.Shard
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a p2p network,
// and a shard transaction pool.
func NewProposer(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, txpool *txpool.TXPool, shardChainDB ethdb.Database, shardID int) (*Proposer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	shard := sharding.NewShard(big.NewInt(int64(shardID)), shardChainDB)
	return &Proposer{
		config,
		client,
		p2p,
		txpool,
		nil,
		shardChainDB,
		shard,
		ctx,
		cancel}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() {
	log.Info("Starting proposer service")
	go p.proposeCollations()
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Info(fmt.Sprintf("Stopping proposer service in shard %d", p.shard.ShardID()))
	defer p.cancel()
	p.txpoolSub.Unsubscribe()
	return nil
}

// proposeCollations listens to the transaction feed and submits collations over an interval.
func (p *Proposer) proposeCollations() {
	feed := p.p2p.Feed(messages.TransactionBroadcast{})
	ch := make(chan p2p.Message, 20) // Is this buffer size too small?
	sub := feed.Subscribe(ch)
	defer sub.Unsubscribe()
	defer close(ch)
	for {
		select {
		case msg := <-ch:
			tx, ok := msg.Data.(messages.TransactionBroadcast)
			if !ok {
				log.Error("Received incorrect p2p message. Wanted a transaction broadcast message")
				break
			}
			log.Info(fmt.Sprintf("Received transaction with hash: %x", tx.Transaction.Hash()))
			if err := p.createCollation(p.ctx, []*types.Transaction{tx.Transaction}); err != nil {
				log.Error(fmt.Sprintf("Create collation failed: %v", err))
			}
		case <-p.ctx.Done():
			log.Error("Proposer context closed, exiting goroutine")
			return
		case err := <-p.txpoolSub.Err():
			log.Error(fmt.Sprintf("Subscriber closed: %v", err))
			return
		}
	}
}

func (p *Proposer) createCollation(ctx context.Context, txs []*types.Transaction) error {
	// Get current block number.
	blockNumber, err := p.client.ChainReader().BlockByNumber(ctx, nil)
	if err != nil {
		return err
	}
	period := new(big.Int).Div(blockNumber.Number(), big.NewInt(p.config.PeriodLength))

	// Create collation.
	collation, err := createCollation(p.client, p.client.Account(), p.client, p.shard.ShardID(), period, txs)
	if err != nil {
		return err
	}

	// Saves the collation to persistent storage in the shardDB.
	if err := p.shard.SaveCollation(collation); err != nil {
		log.Error(fmt.Sprintf("Could not save collation to persistent storage: %v", err))
		return nil
	}

	log.Info(fmt.Sprintf("Saved collation with header hash %v to shardChainDb", collation.Header().Hash().Hex()))

	// Check SMC if we can submit header before addHeader.
	canAdd, err := checkHeaderAdded(p.client, p.shard.ShardID(), period)
	if err != nil {
		return err
	}
	if canAdd {
		AddHeader(p.client, collation)
	}

	return nil
}

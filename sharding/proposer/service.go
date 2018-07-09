// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"math/big"

	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/geth-sharding/sharding/database"
	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
	"github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/syncer"
	"github.com/prysmaticlabs/geth-sharding/sharding/txpool"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
	log "github.com/sirupsen/logrus"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	config    *params.Config
	client    *mainchain.SMCClient
	p2p       *p2p.Server
	txpool    *txpool.TXPool
	txpoolSub event.Subscription
	dbService *database.ShardDB
	shardID   int
	shard     *types.Shard
	ctx       context.Context
	cancel    context.CancelFunc
	sync      *syncer.Syncer
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a p2p network,
// and a shard transaction pool.
func NewProposer(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, txpool *txpool.TXPool, dbService *database.ShardDB, shardID int, sync *syncer.Syncer) (*Proposer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Proposer{
		config,
		client,
		p2p,
		txpool,
		nil,
		dbService,
		shardID,
		nil,
		ctx,
		cancel,
		sync}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() {
	log.Info("Starting proposer service")
	shard := types.NewShard(big.NewInt(int64(p.shardID)), p.dbService.DB())
	p.shard = shard
	go p.proposeCollations()
	go p.sync.HandleCollationBodyRequests(p.shard, p.ctx.Done())
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Warnf("Stopping proposer service in shard %d", p.shard.ShardID())
	defer p.cancel()
	p.txpoolSub.Unsubscribe()
	return nil
}

// proposeCollations listens to the transaction feed and submits collations over an interval.
func (p *Proposer) proposeCollations() {
	requests := make(chan *gethTypes.Transaction)
	p.txpoolSub = p.txpool.TransactionsFeed().Subscribe(requests)
	defer close(requests)
	for {
		select {
		case tx := <-requests:
			log.Infof("Received transaction: %x", tx.Hash())
			if err := p.createCollation(p.ctx, []*gethTypes.Transaction{tx}); err != nil {
				log.Errorf("Create collation failed: %v", err)
			}
		case <-p.ctx.Done():
			log.Info("Proposer context closed, exiting goroutine")
			return
		case <-p.txpoolSub.Err():
			log.Debug("Subscriber closed")
			return
		}
	}
}

func (p *Proposer) createCollation(ctx context.Context, txs []*gethTypes.Transaction) error {
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
		log.Errorf("Could not save collation to persistent storage: %v", err)
		return nil
	}

	log.Infof("Saved collation with header hash %v to shardChainDB", collation.Header().Hash().Hex())

	// Check SMC if we can submit header before addHeader.
	canAdd, err := checkHeaderAdded(p.client, p.shard.ShardID(), period)
	if err != nil {
		return err
	}
	if canAdd {
		AddHeader(p.client, p.client, collation)
	}

	return nil
}

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
	pb "github.com/prysmaticlabs/geth-sharding/proto/sharding/v1"
	"github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/syncer"
	"github.com/prysmaticlabs/geth-sharding/sharding/txpool"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
	"github.com/prysmaticlabs/geth-sharding/shared/legacyutil"
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
	msgChan   chan p2p.Message
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
		nil, // txpoolSub
		dbService,
		shardID,
		nil, // shard
		ctx,
		cancel,
		sync,
		nil, // msgChan
	}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() {
	log.Info("Starting proposer service")
	p.shard = types.NewShard(big.NewInt(int64(p.shardID)), p.dbService.DB())
	p.msgChan = make(chan p2p.Message, 20)
	feed := p.p2p.Feed(pb.Transaction{})
	p.txpoolSub = feed.Subscribe(p.msgChan)
	go p.proposeCollations()
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Warnf("Stopping proposer service in shard %d", p.shard.ShardID())
	defer p.cancel()
	defer close(p.msgChan)
	p.txpoolSub.Unsubscribe()
	return nil
}

// proposeCollations listens to the transaction feed and submits collations over an interval.
func (p *Proposer) proposeCollations() {
	feed := p.p2p.Feed(pb.Transaction{})
	ch := make(chan p2p.Message, 20)
	sub := feed.Subscribe(ch)
	defer sub.Unsubscribe()
	defer close(ch)
	for {
		select {
		case msg := <-ch:
			tx, ok := msg.Data.(*pb.Transaction)
			if !ok {
				log.Error("Received incorrect p2p message. Wanted a transaction broadcast message")
				break
			}
			// log.Debugf("Received transaction: %x", tx)
			if err := p.createCollation(p.ctx, []*gethTypes.Transaction{legacyutil.TransformTransaction(tx)}); err != nil {
				log.Errorf("Create collation failed: %v", err)
			}
		case <-p.ctx.Done():
			log.Debug("Proposer context closed, exiting goroutine")
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

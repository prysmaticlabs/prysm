// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"math/big"

	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/syncer"
	"github.com/prysmaticlabs/prysm/client/txpool"
	"github.com/prysmaticlabs/prysm/client/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/legacyutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "proposer")

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	config    *params.Config
	client    mainchain.FullClient
	p2p       *p2p.Server
	txpool    *txpool.TXPool
	txpoolSub event.Subscription
	dbService *database.DB
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
func NewProposer(config *params.Config, client mainchain.FullClient, p2p *p2p.Server, txpool *txpool.TXPool, dbService *database.DB, shardID int, sync *syncer.Syncer) (*Proposer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Proposer{
		config:    config,
		client:    client,
		p2p:       p2p,
		txpool:    txpool,
		txpoolSub: nil,
		dbService: dbService,
		shardID:   shardID,
		shard:     nil,
		ctx:       ctx,
		cancel:    cancel,
		sync:      sync,
		msgChan:   nil,
	}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() {
	log.Info("Starting proposer service")
	p.shard = types.NewShard(big.NewInt(int64(p.shardID)), p.dbService.DB())
	p.msgChan = make(chan p2p.Message, 20)
	p.txpoolSub = p.p2p.Subscribe(pb.Transaction{}, p.msgChan)
	go p.proposeCollations()
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.WithFields(logrus.Fields{
		"shardID": p.shard.ShardID(),
	}).Warn("Stopping proposer service")
	defer p.cancel()
	defer close(p.msgChan)
	p.txpoolSub.Unsubscribe()
	return nil
}

// proposeCollations listens to the transaction feed and submits collations over an interval.
func (p *Proposer) proposeCollations() {
	ch := make(chan p2p.Message, 20)
	sub := p.p2p.Subscribe(pb.Transaction{}, ch)
	collation := []*gethTypes.Transaction{}
	sizeOfCollation := int64(0)
	period, err := p.currentPeriod(p.ctx)
	if err != nil {
		log.Errorf("Unable to get current period: %v", err)
	}

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
			log.Debugf("Received transaction: %x", tx)
			gethtx := legacyutil.TransformTransaction(tx)
			currentperiod, err := p.currentPeriod(p.ctx)
			if err != nil {
				log.Errorf("Unable to get current period: %v", err)
			}

			// This checks for when the size of transactions is equal to or slightly less than the CollationSizeLimit
			// and if the current period has changed so as to know when to create collations with the received transactions.
			if (sizeOfCollation+int64(gethtx.Size())) > p.config.CollationSizeLimit || period.Cmp(currentperiod) != 0 {
				if err := p.createCollation(p.ctx, collation); err != nil {
					log.Errorf("Create collation failed: %v", err)
					return
				}
				collation = []*gethTypes.Transaction{}
				sizeOfCollation = 0
				log.Info("Collation created")

				if period.Cmp(currentperiod) != 0 {
					_ = period.Set(currentperiod)
				}
			}

			collation = append(collation, legacyutil.TransformTransaction(tx))
			sizeOfCollation += int64(gethtx.Size())
		case <-p.ctx.Done():
			log.Debug("Proposer context closed, exiting goroutine")
			return
		case <-p.txpoolSub.Err():
			log.Debug("Subscriber closed")
			return
		}
	}
}

func (p *Proposer) currentPeriod(ctx context.Context) (*big.Int, error) {

	// Get current block number.
	blockNumber, err := p.client.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	period := new(big.Int).Div(blockNumber.Number(), big.NewInt(p.config.PeriodLength))

	return period, nil

}

func (p *Proposer) createCollation(ctx context.Context, txs []*gethTypes.Transaction) error {

	period, err := p.currentPeriod(ctx)
	if err != nil {
		return err
	}

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

	log.WithFields(logrus.Fields{
		"headerHash": collation.Header().Hash().Hex(),
	}).Info("Saved collation to shardChainDB")

	// Check SMC if we can submit header before addHeader.
	canAdd, err := checkHeaderAdded(p.client, p.shard.ShardID(), period)
	if err != nil {
		return err
	}
	if canAdd {
		if err := AddHeader(p.client, p.client, collation); err != nil {
			return err
		}
	}

	return nil
}

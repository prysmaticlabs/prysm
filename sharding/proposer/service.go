// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
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
	shardChainDb ethdb.Database
	shard        *sharding.Shard
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a p2p network,
// and a shard transaction pool.
func NewProposer(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, txpool *txpool.TXPool, shardChainDb ethdb.Database, shardID int) (*Proposer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	shard := sharding.NewShard(big.NewInt(int64(shardID)), shardChainDb)
	return &Proposer{config, client, p2p, txpool, shardChainDb, shard, ctx, cancel}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() {
	log.Info("Starting proposer service")
	go p.proposeCollations()

	feed, err := p.p2p.Feed(sharding.CollationBodyRequest{})
	if err != nil {
		log.Error(fmt.Sprintf("Could not initialize p2p feed: %v", err))
		return
	}

	go p.handleCollationBodyRequests(feed)
	go p.simulateNotaryRequests(feed)
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() {
	// Triggers a cancel call in the actor's context which shuts down every goroutine
	// in this service.
	defer p.cancel()
	log.Warn(fmt.Sprintf("Stopping proposer service in shard %d", p.shard.ShardID()))
}

// proposeCollations is the main event loop of a proposer service that listens for
// incoming transactions and adds them to the SMC.
func (p *Proposer) proposeCollations() {
	// TODO: Receive TXs from shard TX generator or TXpool (Github Issues 153 and 161)
	var txs []*types.Transaction
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		txs = append(txs, types.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}

	// Get current block number.
	blockNumber, err := p.client.ChainReader().BlockByNumber(context.Background(), nil)
	if err != nil {
		log.Error(fmt.Sprintf("Could not fetch current block number: %v", err))
		return
	}
	period := new(big.Int).Div(blockNumber.Number(), big.NewInt(p.config.PeriodLength))

	// Create collation.
	collation, err := createCollation(p.client, p.shard.ShardID(), period, txs)
	if err != nil {
		log.Error(fmt.Sprintf("Could not create collation: %v", err))
		return
	}

	// Saves the collation to persistent storage in the shardDB.
	if err := p.shard.SaveCollation(collation); err != nil {
		log.Error(fmt.Sprintf("Could not save collation to persistent storage: %v", err))
		return
	}

	log.Info(fmt.Sprintf("Saved collation with header hash %v to shardChainDb", collation.Header().Hash().Hex()))

	// Check SMC if we can submit header before addHeader.
	canAdd, err := checkHeaderAdded(p.client, p.shard.ShardID(), period)
	if err != nil {
		log.Error(fmt.Sprintf("Could not check if we can submit header: %v", err))
		return
	}
	if canAdd {
		addHeader(p.client, collation)
	}
}

// handleCollationBodyRequests subscribes to messages from the shardp2p
// network and responds to a specific peer that requested the body using
// the feed exposed by the p2p server's API.
func (p *Proposer) handleCollationBodyRequests(feed *event.Feed) {

	ch := make(chan p2p.Message, 100)
	sub := feed.Subscribe(ch)

	defer sub.Unsubscribe()
	defer close(ch)

	for {
		select {
		case <-p.ctx.Done():
			return
		case req := <-ch:
			log.Info(fmt.Sprintf("Received p2p request from notary: %v", req))
			res, err := collationBodyResponse(req, p.client, p.shard)
			if err != nil {
				log.Error("Could not construct response: %v", err)
				continue
			}
			log.Info(fmt.Sprintf("Responding to p2p request with collation with headerHash: %v", res.HeaderHash.Hex()))
			// Reply to that specific peer only.
			// TODO: Implement this and see the response from the other end.
			p.p2p.Send(res, req.Peer)

		case err := <-sub.Err():
			log.Error("Subscriber failed: %v", err)
			return
		}
	}
}

func (p *Proposer) simulateNotaryRequests(feed *event.Feed) {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			req, err := constructNotaryRequest(p.client, p.client, p.shard.ShardID(), p.config.PeriodLength)
			if err != nil {
				log.Error(fmt.Sprintf("Error constructing collation body request: %v, trying again...", err))
				continue
			}
			msg := p2p.Message{
				Peer: p2p.Peer{},
				Data: req,
			}
			feed.Send(msg)
			log.Info("Sent request for collation body via a shardp2p feed")
		}
	}
}

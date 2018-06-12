// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	client   *mainchain.SMCClient
	shardp2p *p2p.Server
	txpool   sharding.TXPool
	shard    *sharding.Shard
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a shardp2p network,
// and a shard transaction pool.
func NewProposer(client *mainchain.SMCClient, shardp2p *p2p.Server, txpool sharding.TXPool, shardChainDb ethdb.Database, shardID int) (*Proposer, error) {
	shard := sharding.NewShard(big.NewInt(int64(shardID)), shardChainDb)
	return &Proposer{client, shardp2p, txpool, shard}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() error {
	log.Info("Starting proposer service")
	go p.proposeCollations()
	go p.handleCollationBodyRequests()
	go simulateNotaryRequests(p.client, p.shardp2p, p.shard.ShardID())
	return nil
}

// handleCollationBodyRequests subscribes to messages from the shardp2p
// network and responds to a specific peer that requested the body using
// the feed exposed by the p2p server's API.
func (p *Proposer) handleCollationBodyRequests() {
	feed, err := p.shardp2p.Feed(sharding.CollationBodyRequest{})
	if err != nil {
		log.Error(fmt.Sprintf("Could not initialize p2p feed: %v", err))
		return
	}
	ch := make(chan p2p.Message, 100)
	sub := feed.Subscribe(ch)
	// TODO: close chan and unsubscribe in Stop().

	// Set up a context with deadline or timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case req := <-ch:
			log.Info("Received p2p request from notary")

			// Type assertion helps us catch incorrect data requests.
			msg, ok := req.Data.(sharding.CollationBodyRequest)
			if !ok {
				log.Error(fmt.Sprintf("Received incorrect data request type: %v", msg))
			}

			// Reply to that specific peer only.
			collation, err := p.shard.CollationByHeaderHash(msg.HeaderHash)
			if err != nil {
				log.Error(fmt.Sprintf("Could not fetch collation: %v", err))
			}

			res := &sharding.CollationBodyResponse{HeaderHash: msg.HeaderHash, Body: collation.Body()}
			p.shardp2p.Send(res, req.Peer)

		case <-ctx.Done():
			log.Error("Subscriber timed out")
		case err := <-sub.Err():
			log.Error("Subscriber failed: %v", err)
			return
		}
	}
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
	period := new(big.Int).Div(blockNumber.Number(), big.NewInt(sharding.PeriodLength))

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

	log.Info("Saved collation with header hash %v to shardChainDb", collation.Header().Hash().Hex())

	// Check SMC if we can submit header before addHeader.
	canAdd, err := checkHeaderAdded(p.client, p.shard.ShardID(), period)
	if err != nil {
		log.Error(fmt.Sprintf("Could not check if we can submit header: %v", err))
		return
	}
	if canAdd {
		addHeader(p.client, collation)
	}

	return
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Info(fmt.Sprintf("Stopping proposer service in shard %d", p.shard.ShardID()))
	return nil
}

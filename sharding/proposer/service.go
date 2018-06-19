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
	config *params.Config
	client *mainchain.SMCClient
	p2p    *p2p.Server
	txpool *txpool.TXPool
	shard  *sharding.Shard
	ctx    context.Context
	cancel context.CancelFunc
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a config, a mainchain client, a p2p server,
// a shardChainDb, and a shard transaction pool.
func NewProposer(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, txpool *txpool.TXPool, shardChainDb ethdb.Database, shardID int) (*Proposer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	shard := sharding.NewShard(big.NewInt(int64(shardID)), shardChainDb)
	return &Proposer{config, client, p2p, txpool, shard, ctx, cancel}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() {
	log.Info("Starting proposer service")
	go p.proposeCollations()
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	// Triggers a cancel call in the actor's context which shuts down every goroutine
	// in this service.
	defer p.cancel()
	log.Warn(fmt.Sprintf("Stopping proposer service in shard %d", p.shard.ShardID()))
	return nil
}

// proposeCollations is the main event loop of a proposer service that listens for
// incoming transactions and adds them to the SMC.
func (p *Proposer) proposeCollations() {
	// TODO: Receive TXs from shard TX generator or TXpool (Github Issues 153 and 161)
	// TODO: Do this as a loop that runs on every period using a similar pattern to other
	// goroutines in this file.
	// for {
	//	select {
	// 	case <-p.ctx.Done():
	//		return
	//  default:
	//			(logic goes here)
	//  }
	// }
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
	collation, err := createCollation(p.client, p.client.Account(), p.client, p.shard.ShardID(), period, txs)
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
		AddHeader(p.client, collation)
	}
}

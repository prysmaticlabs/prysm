// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"crypto/rand"
	"math/big"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
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
	shardChainDb ethdb.Database
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a shardp2p network,
// and a shard transaction pool.
func NewProposer(client *mainchain.SMCClient, shardp2p sharding.ShardP2P, txpool sharding.TXPool, shardChainDb ethdb.Database) (*Proposer, error) {
	// Initializes a  directory persistent db.
	return &Proposer{client, shardp2p, txpool, shardChainDb}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() error {
	log.Info("Starting proposer service")

	go p.subscribeTransactions()

	// TODO: Receive TXs from shard TX generator or TXpool (Github Issues 153 and 161)
	var txs []*types.Transaction
	for i := 0; i < 10; i++ {
		data := make([]byte, 1024)
		rand.Read(data)
		txs = append(txs, types.NewTransaction(0, common.HexToAddress("0x0"),
			nil, 0, nil, data))
	}

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
	return nil
}

func (p *Proposer) subscribeTransactions() {
	// Subscribes to incoming transactions from the txpool via the shardp2p network.
	for {
		subchan := make(chan int)
		sub := p.txpool.TransactionsFeed().Subscribe(subchan)
		// 10 second time out for the subscription.
		timeout := time.NewTimer(10 * time.Second)
		select {
		case v := <-subchan:
			log.Info(fmt.Sprintf("Received transaction with id: %d", v))
		case <-timeout.C:
			log.Error("Receive timeout")
		}

		sub.Unsubscribe()
		select {
		case _, ok := <-sub.Err():
			if ok {
				log.Error("Channel not closed after unsubscribe")
			}
		case <-timeout.C:
			log.Error("Unsubscribe timeout")
		}
	}
}

// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
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
	requests     chan *types.Transaction
	txpoolSub    event.Subscription
	shardChainDb ethdb.Database
	shardID      int
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a p2p network,
// and a shard transaction pool.
func NewProposer(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, txpool *txpool.TXPool, shardChainDb ethdb.Database, shardID int) (*Proposer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Proposer{
		config,
		client,
		p2p,
		txpool,
		make(chan *types.Transaction),
		nil,
		shardChainDb,
		shardID,
		ctx,
		cancel}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() {
	log.Info(fmt.Sprintf("Starting proposer service in shard %d", p.shardID))
	p.subscribeFeed()
	go p.proposeCollations(p.ctx.Done(), p.txpoolSub.Err(), p.requests, p.client.Account(), p.client.ChainReader(), p.client, p.client, p.client)
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Info(fmt.Sprintf("Stopping proposer service in shard %d", p.shardID))
	defer p.cancel()
	p.txpoolSub.Unsubscribe()
	return nil
}

func (p *Proposer) subscribeFeed() {
	p.txpoolSub = p.txpool.TransactionsFeed().Subscribe(p.requests)
}

// proposeCollations listens to the transaction feed and submits collations over an interval.
func (p *Proposer) proposeCollations(done <-chan struct{}, subErr <-chan error, requests <-chan *types.Transaction, account *accounts.Account,
	reader mainchain.Reader, caller mainchain.ContractCaller, signer mainchain.Signer, transactor mainchain.ContractTransactor) {
	for {
		select {
		case tx := <-requests:
			log.Info(fmt.Sprintf("Received transaction: %x", tx.Hash()))
			err := p.submitCollationHeader([]*types.Transaction{tx}, big.NewInt(p.config.PeriodLength),
				big.NewInt(int64(p.shardID)), account, reader, caller, signer, transactor)
			if err != nil {
				log.Error(fmt.Sprintf("Create collation failed: %v", err))
			}
		case <-done:
			log.Error("Proposer context closed, exiting goroutine")
			return
		case <-subErr:
			log.Error(fmt.Sprintf("Transaction feed subscriber closed"))
			return
		}
	}
}

func (p *Proposer) submitCollationHeader(txs []*types.Transaction, periodLength *big.Int, shardID *big.Int, account *accounts.Account,
	reader mainchain.Reader, caller mainchain.ContractCaller, signer mainchain.Signer, transactor mainchain.ContractTransactor) error {
	period, err := getPeriod(reader, periodLength)
	if err != nil {
		return err
	}

	if err := submitCollationHeader(txs, period, shardID, account, caller, signer, transactor); err != nil {
		return err
	}

	return nil
}

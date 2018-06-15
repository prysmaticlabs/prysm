package simulator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/syncer"
)

// Simulator is a service in a shard node
// that simulates requests from remote notes coming over
// the shardp2p network. For example, if we are running a
// proposer service, we would want to simulate notary requests
// coming to us via a p2p feed. This service will be removed once
// p2p internals and end-to-end testing across remote nodes have been
// implemented.
type Simulator struct {
	config  *params.Config
	client  *mainchain.SMCClient
	p2p     *p2p.Server
	shardID int
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewSimulator creates a struct instance of a simulator service.
// It will have access to config, a mainchain client, a p2p server,
// and a shardID.
func NewSimulator(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, shardID int) *Simulator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Simulator{config, client, p2p, shardID, ctx, cancel}
}

// Start the main loop for simulating p2p requests.
func (s *Simulator) Start() {
	log.Info("Starting simulator service")
	feed, err := s.p2p.Feed(sharding.CollationBodyRequest{})
	if err != nil {
		log.Error(fmt.Sprintf("Could not initialize p2p feed: %v", err))
		return
	}
	go s.simulateNotaryRequests(feed)
}

// Stop the main loop for simulator requests.
func (s *Simulator) Stop() error {
	// Triggers a cancel call in the service's context which shuts down every goroutine
	// in this service.
	defer s.cancel()
	log.Warn("Stopping simulator service")
	return nil
}

// simulateNotaryRequests simulates p2p message sent out by notaries
// once the system is in production. Notaries will be performing
// this action within their own service when they are selected on a shard, period
// pair to perform their responsibilities. This function in particular simulates
// requests for collation bodies that will be relayed to the appropriate proposer
// by the p2p feed layer.
func (s *Simulator) simulateNotaryRequests(feed *event.Feed) {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			blockNumber, err := s.client.ChainReader().BlockByNumber(s.ctx, nil)
			if err != nil {
				log.Error(fmt.Sprintf("Could not fetch current block number: %v", err))
				continue
			}

			period := new(big.Int).Div(blockNumber.Number(), big.NewInt(s.config.PeriodLength))
			req, err := syncer.RequestCollationBody(s.client, big.NewInt(int64(s.shardID)), period)
			if err != nil {
				log.Error(fmt.Sprintf("Error constructing collation body request: %v, trying again...", err))
				continue
			}
			if req == nil {
				continue
			}
			msg := p2p.Message{
				Peer: p2p.Peer{},
				Data: *req,
			}
			feed.Send(msg)
			log.Info("Sent request for collation body via a shardp2p feed")
		}
	}
}

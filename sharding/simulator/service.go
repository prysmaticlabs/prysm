package simulator

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/p2p/messages"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/syncer"
	"github.com/ethereum/go-ethereum/sharding/utils"
)

// Simulator is a service in a shard node that simulates requests from
// remote notes coming over the shardp2p network. For example, if
// we are running a proposer service, we would want to simulate notary requests
// requests coming to us via a p2p feed. This service will be removed
// once p2p internals and end-to-end testing across remote
// nodes have been implemented.
type Simulator struct {
	config  *params.Config
	client  *mainchain.SMCClient
	p2p     *p2p.Server
	shardID int
	ctx     context.Context
	cancel  context.CancelFunc
	errChan chan error    // Useful channel for handling errors at the service layer.
	delay   time.Duration // The delay (in seconds) between simulator requests sent via p2p.
}

// NewSimulator creates a struct instance of a simulator service.
// It will have access to config, a mainchain client, a p2p server,
// and a shardID.
func NewSimulator(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, shardID int, delay time.Duration) (*Simulator, error) {
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	return &Simulator{config, client, p2p, shardID, ctx, cancel, errChan, delay}, nil
}

// Start the main loop for simulating p2p requests.
func (s *Simulator) Start() {
	log.Info("Starting simulator service")
	feed := s.p2p.Feed(messages.CollationBodyRequest{})
	periodLength := big.NewInt(s.config.PeriodLength)
	shardID := big.NewInt(int64(s.shardID))
	go utils.HandleServiceErrors(s.ctx.Done(), s.errChan)
	go simulateNotaryRequests(s.client.SMCCaller(), s.client.ChainReader(), feed, periodLength, shardID, s.ctx.Done(), time.After(time.Second*s.delay), s.errChan)
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
func simulateNotaryRequests(fetcher mainchain.RecordFetcher, reader mainchain.Reader, feed *event.Feed, periodLength *big.Int, shardID *big.Int, done <-chan struct{}, delay <-chan time.Time, errChan chan<- error) {
	for {
		select {
		// Makes sure to close this goroutine when the service stops.
		case <-done:
			return
		case <-delay:
			blockNumber, err := reader.BlockByNumber(context.Background(), nil)
			if err != nil {
				errChan <- fmt.Errorf("could not fetch current block number: %v", err)
				continue
			}

			period := new(big.Int).Div(blockNumber.Number(), periodLength)
			req, err := syncer.RequestCollationBody(fetcher, shardID, period)
			if err != nil {
				errChan <- fmt.Errorf("error constructing collation body request: %v", err)
				continue
			}
			if req != nil {
				msg := p2p.Message{
					Peer: p2p.Peer{},
					Data: *req,
				}
				feed.Send(msg)
				log.Info("Sent request for collation body via a shardp2p feed")
			}
		}
	}
}

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
)

// Simulator is a service in a shard node
// that simulates requests from remote notes coming over
// the shardp2p network. For example, if we are running a
// proposer service, we would want to simulate notary requests
// coming to us via a p2p feed. This service will be removed once
// p2p internals and end-to-end testing across remote nodes have been
// implemented.
type Simulator struct {
	config      *params.Config
	client      *mainchain.SMCClient
	p2p         *p2p.Server
	shardID     int
	ctx         context.Context
	cancel      context.CancelFunc
	errChan     chan error       // Useful channel for handling errors at the service layer.
	requestSent chan interface{} // Useful channel for processing logic upon a request being sent via p2p.
}

// NewSimulator creates a struct instance of a simulator service.
// It will have access to config, a mainchain client, a p2p server,
// and a shardID.
func NewSimulator(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, shardID int) (*Simulator, error) {
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	requestSent := make(chan interface{})
	return &Simulator{config, client, p2p, shardID, ctx, cancel, errChan, requestSent}, nil
}

// Start the main loop for simulating p2p requests.
func (s *Simulator) Start() {
	log.Info("Starting simulator service")
	feed := s.p2p.Feed(messages.CollationBodyRequest{})
	go s.simulateNotaryRequests(s.client.SMCCaller(), s.client.ChainReader(), feed)
	go s.handleServiceErrors()
}

// Stop the main loop for simulator requests.
func (s *Simulator) Stop() error {
	// Triggers a cancel call in the service's context which shuts down every goroutine
	// in this service.
	defer s.cancel()
	log.Warn("Stopping simulator service")
	return nil
}

// handleServiceErrors manages a goroutine that listens for errors broadcast to
// this service's error channel. This serves as a final step for error logging
// and is stopped upon the service shutting down.
func (s *Simulator) handleServiceErrors() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case err := <-s.errChan:
			log.Error(fmt.Sprintf("Simulator service error: %v", err))
		}
	}
}

// simulateNotaryRequests simulates p2p message sent out by notaries
// once the system is in production. Notaries will be performing
// this action within their own service when they are selected on a shard, period
// pair to perform their responsibilities. This function in particular simulates
// requests for collation bodies that will be relayed to the appropriate proposer
// by the p2p feed layer.
func (s *Simulator) simulateNotaryRequests(fetcher mainchain.RecordFetcher, reader mainchain.Reader, feed *event.Feed) {
	for {
		select {
		// Makes sure to close this goroutine when the service stops.
		case <-s.ctx.Done():
			return
		case <-time.After(time.Second * 5):
			blockNumber, err := reader.BlockByNumber(s.ctx, nil)
			if err != nil {
				s.errChan <- fmt.Errorf("could not fetch current block number: %v", err)
				continue
			}

			period := new(big.Int).Div(blockNumber.Number(), big.NewInt(s.config.PeriodLength))
			req, err := syncer.RequestCollationBody(fetcher, big.NewInt(int64(s.shardID)), period)
			if err != nil {
				s.errChan <- fmt.Errorf("error constructing collation body request: %v", err)
				continue
			}
			if req != nil {
				msg := p2p.Message{
					Peer: p2p.Peer{},
					Data: *req,
				}
				// feed.Send(msg)
				s.p2p.Broadcast(req)

				// Notifies the requestSent channel for any other handlers that could run upon
				// this event occurring (also useful for tests.)
				//s.requestSent <- msg
				_ = msg

				log.Info("Sent request for collation body via a shardp2p feed")
			}
		}
	}
}

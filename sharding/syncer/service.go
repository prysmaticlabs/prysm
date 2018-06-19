package syncer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/p2p/messages"
	"github.com/ethereum/go-ethereum/sharding/params"
)

// Syncer represents a service that provides handlers for shard chain
// data requests/responses between remote nodes and event loops for
// performing windback sync across nodes, handling reorgs, and synchronizing
// items such as transactions and in future sharding iterations: state.
type Syncer struct {
	config       *params.Config
	client       *mainchain.SMCClient
	shard        *sharding.Shard
	p2p          *p2p.Server
	ctx          context.Context
	cancel       context.CancelFunc
	errChan      chan error       // Useful channel for handling errors at the service layer.
	responseSent chan interface{} // Useful channel for processing logic upon a response being sent via p2p.
}

// NewSyncer creates a struct instance of a syncer service.
// It will have access to config, a mainchain client, a p2p server,
// a shardChainDb, and a shardID.
func NewSyncer(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, shardChainDb ethdb.Database, shardID int) (*Syncer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	shard := sharding.NewShard(big.NewInt(int64(shardID)), shardChainDb)
	errChan := make(chan error)
	responseSent := make(chan interface{})
	return &Syncer{config, client, shard, p2p, ctx, cancel, errChan, responseSent}, nil
}

// Start the main loop for handling shard chain data requests.
func (s *Syncer) Start() {
	log.Info("Starting sync service")
	go s.handleCollationBodyRequests(s.client, s.p2p.Feed(messages.CollationBodyRequest{}))
	go s.handleServiceErrors()
}

// Stop the main loop.
func (s *Syncer) Stop() error {
	// Triggers a cancel call in the service's context which shuts down every goroutine
	// in this service.
	defer s.cancel()
	log.Warn("Stopping sync service")
	return nil
}

// handleServiceErrors manages a goroutine that listens for errors broadcast to
// this service's error channel. This serves as a final step for error logging
// and is stopped upon the service shutting down.
func (s *Syncer) handleServiceErrors() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case err := <-s.errChan:
			log.Error(fmt.Sprintf("Sync service error: %v", err))
		}
	}
}

// handleCollationBodyRequests subscribes to messages from the shardp2p
// network and responds to a specific peer that requested the body using
// the Send method exposed by the p2p server's API (implementing the p2p.Sender interface).
func (s *Syncer) handleCollationBodyRequests(signer mainchain.Signer, feed *event.Feed) {

	ch := make(chan p2p.Message, 100)
	sub := feed.Subscribe(ch)

	defer sub.Unsubscribe()

	for {
		select {
		// Makes sure to close this goroutine when the service stops.
		case <-s.ctx.Done():
			return
		case req := <-ch:
			if req.Data != nil {
				log.Info(fmt.Sprintf("Received p2p request of type: %T", req))
				res, err := RespondCollationBody(req, signer, s.shard)
				if err != nil {
					s.errChan <- fmt.Errorf("could not construct response: %v", err)
					continue
				}
				log.Info(fmt.Sprintf("Responding to p2p request with collation with headerHash: %v", res.HeaderHash.Hex()))

				// Notifies the response sent channel for any other handlers that could run upon
				// this event occurring (also useful for tests.)
				s.responseSent <- res

				// Reply to that specific peer only.
				s.p2p.Send(*res, req.Peer)
			}
		}
	}
}

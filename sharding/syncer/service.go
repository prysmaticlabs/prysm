package syncer

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/p2p/messages"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/utils"
)

// Syncer represents a service that provides handlers for shard chain
// data requests/responses between remote nodes and event loops for
// performing windback sync across nodes, handling reorgs, and synchronizing
// items such as transactions and in future sharding iterations: state.
type Syncer struct {
	config       *params.Config
	client       *mainchain.SMCClient
	shardID      int
	shardChainDB *database.ShardDB
	p2p          *p2p.Server
	ctx          context.Context
	cancel       context.CancelFunc
	msgChan      chan p2p.Message
	bodyRequests event.Subscription
	errChan      chan error // Useful channel for handling errors at the service layer.
}

// NewSyncer creates a struct instance of a syncer service.
// It will have access to config, a signer, a p2p server,
// a shardChainDB, and a shardID.
func NewSyncer(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, shardChainDB *database.ShardDB, shardID int) (*Syncer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	return &Syncer{config, client, shardID, shardChainDB, p2p, ctx, cancel, nil, nil, errChan}, nil
}

// Start the main loop for handling shard chain data requests.
func (s *Syncer) Start() {
	log.Info("Starting sync service")

	shard := sharding.NewShard(big.NewInt(int64(s.shardID)), s.shardChainDB.DB())

	s.msgChan = make(chan p2p.Message, 100)
	s.bodyRequests = s.p2p.Feed(messages.CollationBodyRequest{}).Subscribe(s.msgChan)
	go s.HandleCollationBodyRequests(shard)
	go utils.HandleServiceErrors(s.ctx.Done(), s.errChan)
}

// Stop the main loop.
func (s *Syncer) Stop() error {
	// Triggers a cancel call in the service's context which shuts down every goroutine
	// in this service.
	defer s.cancel()
	defer close(s.errChan)
	defer close(s.msgChan)
	log.Warn("Stopping sync service")
	s.bodyRequests.Unsubscribe()
	return nil
}

// HandleCollationBodyRequests subscribes to messages from the shardp2p
// network and responds to a specific peer that requested the body using
// the Send method exposed by the p2p server's API (implementing the p2p.Sender interface).
func (s *Syncer) HandleCollationBodyRequests(collationFetcher sharding.CollationFetcher) {
	for {
		select {
		// Makes sure to close this goroutine when the service stops.
		case <-s.ctx.Done():
			return
		case req := <-s.msgChan:
			if req.Data != nil {
				log.Info(fmt.Sprintf("Received p2p request of type: %T", req))
				res, err := RespondCollationBody(req, collationFetcher)
				if err != nil {
					s.errChan <- fmt.Errorf("could not construct response: %v", err)
					continue
				}

				// Reply to that specific peer only.
				s.p2p.Send(*res, req.Peer)
				log.Info(fmt.Sprintf("Responding to p2p request with collation with headerHash: %v", res.HeaderHash.Hex()))
			}
		case <-s.bodyRequests.Err():
			s.errChan <- errors.New("subscriber failed")
			return
		}
	}
}

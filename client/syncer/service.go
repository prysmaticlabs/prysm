package syncer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

// Syncer represents a service that provides handlers for shard chain
// data requests/responses between remote nodes and event loops for
// performing windback sync across nodes, handling reorgs, and synchronizing
// items such as transactions and in future sharding iterations: state.
type Syncer struct {
	config           *params.Config
	client           *mainchain.SMCClient
	shardID          int
	db               *database.DB
	p2p              *p2p.Server
	ctx              context.Context
	cancel           context.CancelFunc
	collationFetcher types.CollationFetcher
	collationBodyBuf chan p2p.Message
}

// NewSyncer creates a struct instance of a syncer service.
// It will have access to config, a signer, a p2p server,
// a shardChainDB, and a shardID.
func NewSyncer(config *params.Config, client *mainchain.SMCClient, shardp2p *p2p.Server, db *database.DB, shardID int) (*Syncer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Syncer{
		config:           config,
		client:           client,
		shardID:          shardID,
		db:               db,
		p2p:              shardp2p,
		ctx:              ctx,
		cancel:           cancel,
		collationBodyBuf: make(chan p2p.Message, 100),
	}, nil
}

// Start the main loop for handling shard chain data requests.
func (s *Syncer) Start() {
	log.Info("Starting sync service")
	s.collationFetcher = types.NewShard(big.NewInt(int64(s.shardID)), s.db.DB())
	go s.run(s.ctx.Done())
}

// Stop the main loop.
func (s *Syncer) Stop() error {
	defer s.cancel()
	log.Info("Stopping sync service")
	return nil
}

func (s *Syncer) run(done <-chan struct{}) {
	// collationBodySub subscribes to messages from the shardp2p
	// network and responds to a specific peer that requested the body using
	// the Send method exposed by the p2p server's API.
	collationBodySub := s.p2p.Subscribe(pb.CollationBodyRequest{}, s.collationBodyBuf)
	defer collationBodySub.Unsubscribe()

	for {
		select {
		// Makes sure to close this goroutine when the service stops.
		case <-done:
			return

		case req := <-s.collationBodyBuf:
			if req.Data != nil {
				log.Debugf("Received p2p request of type: %T", req.Data)
				res, err := RespondCollationBody(req, s.collationFetcher)
				if err != nil {
					log.Errorf("Could not construct response: %v", err)
					continue
				}

				if res == nil {
					// TODO: Send that we don't have it?
					log.Debug("No response for this collation request. Not sending anything.")
					continue
				}

				// Reply to that specific peer only.
				s.p2p.Send(res, req.Peer)
				log.WithFields(logrus.Fields{
					"headerHash": fmt.Sprintf("0x%v", common.Bytes2Hex(res.HeaderHash)),
				}).Info("Responding to p2p collation request")
			}
		case <-collationBodySub.Err():
			log.Debugf("Subscriber failed")
			return
		}
	}
}

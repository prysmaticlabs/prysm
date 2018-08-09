// Package observer launches a service attached to the sharding node
// that simply observes activity across the sharded Ethereum network.
package observer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/syncer"
	"github.com/prysmaticlabs/prysm/client/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "observer")

// Observer holds functionality required to run an observer service
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Observer struct {
	p2p              *p2p.Server
	dbService        *database.DB
	shardID          int
	ctx              context.Context
	cancel           context.CancelFunc
	sync             *syncer.Syncer
	client           *mainchain.SMCClient
	collationFetcher types.CollationFetcher
	collationBodyBuf chan p2p.Message
}

// NewObserver creates a struct instance of a observer service,
// it will have access to a p2p server and a shardChainDB.
func NewObserver(shardp2p *p2p.Server, dbService *database.DB, shardID int, sync *syncer.Syncer, client *mainchain.SMCClient) (*Observer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Observer{
		p2p:              shardp2p,
		dbService:        dbService,
		shardID:          shardID,
		ctx:              ctx,
		cancel:           cancel,
		sync:             sync,
		client:           client,
		collationBodyBuf: make(chan p2p.Message, 100),
	}, nil
}

// Start the main loop for observer service.
func (o *Observer) Start() {
	log.Info("Starting observer service")
	o.collationFetcher = types.NewShard(big.NewInt(int64(o.shardID)), o.dbService.DB())
	go o.run(o.ctx.Done())
}

// Stop the main loop for observer service.
func (o *Observer) Stop() error {

	defer o.cancel()
	log.Info("Stopping observer service")
	return nil
}

func (o *Observer) run(done <-chan struct{}) {
	// collationBodySub subscribes to messages from the shardp2p
	// network and responds to a specific peer that requested the body using
	// the Send method exposed by the p2p server's API.
	collationBodySub := o.p2p.Subscribe(pb.CollationBodyRequest{}, o.collationBodyBuf)
	defer collationBodySub.Unsubscribe()

	for {
		select {
		case <-done:
			log.Debug("Observer context closed, exiting goroutine")
			return

		case req := <-o.collationBodyBuf:
			if req.Data != nil {
				log.Debugf("Received p2p request of type: %T", req.Data)
				res, err := syncer.RespondCollationBody(req, o.collationFetcher)
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
				o.p2p.Send(res, req.Peer)
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

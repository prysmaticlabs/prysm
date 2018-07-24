// Package observer launches a service attached to the sharding node
// that simply observes activity across the sharded Ethereum network.
package observer

import (
	"context"
	"math/big"

	"github.com/prysmaticlabs/prysm/client/database"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/syncer"
	"github.com/prysmaticlabs/prysm/client/types"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "observer")

// Observer holds functionality required to run an observer service
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Observer struct {
	p2p       *p2p.Server
	dbService *database.ShardDB
	shardID   int
	shard     *types.Shard
	ctx       context.Context
	cancel    context.CancelFunc
	sync      *syncer.Syncer
	client    *mainchain.SMCClient
}

// NewObserver creates a struct instance of a observer service,
// it will have access to a p2p server and a shardChainDB.
func NewObserver(p2p *p2p.Server, dbService *database.ShardDB, shardID int, sync *syncer.Syncer, client *mainchain.SMCClient) (*Observer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Observer{p2p, dbService, shardID, nil, ctx, cancel, sync, client}, nil
}

// Start the main loop for observer service.
func (o *Observer) Start() {
	log.Info("Starting observer service")
	o.shard = types.NewShard(big.NewInt(int64(o.shardID)), o.dbService.DB())
	go o.sync.HandleCollationBodyRequests(o.shard, o.ctx.Done())
}

// Stop the main loop for observer service.
func (o *Observer) Stop() error {
	// Triggers a cancel call in the service's context which shuts down every goroutine
	// in this service.
	defer o.cancel()
	log.Info("Stopping observer service")
	return nil
}

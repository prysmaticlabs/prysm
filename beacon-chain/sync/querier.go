package sync

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var queryLog = logrus.WithField("prefix", "syncQuerier")

// QuerierConfig defines the configurable properties of SyncQuerier.
type QuerierConfig struct {
	ResponseBufferSize int
	P2P                p2pAPI
	BeaconDB           *db.BeaconDB
	CurentHeadSlot     uint64
}

// DefaultQuerierConfig provides the default configuration for a sync service.
// ResponseBufferSize determines that buffer size of the `responseBuf` channel.
func DefaultQuerierConfig() *QuerierConfig {
	return &QuerierConfig{
		ResponseBufferSize: 100,
	}
}

// Querier defines the main class in this package.
// See the package comments for a general description of the service's functions.
type Querier struct {
	ctx             context.Context
	cancel          context.CancelFunc
	p2p             p2pAPI
	db              *db.BeaconDB
	curentHeadSlot  uint64
	currentHeadHash []byte
	responseBuf     chan p2p.Message
}

// NewQuerierService constructs a new Sync Querier Service.
// This method is normally called by the main node.
func NewQuerierService(ctx context.Context,
	cfg *QuerierConfig,
) *Querier {
	ctx, cancel := context.WithCancel(ctx)

	responseBuf := make(chan p2p.Message, cfg.ResponseBufferSize)

	return &Querier{
		ctx:            ctx,
		cancel:         cancel,
		p2p:            cfg.P2P,
		db:             cfg.BeaconDB,
		responseBuf:    responseBuf,
		curentHeadSlot: cfg.CurentHeadSlot,
	}
}

// Start begins the goroutine.
func (q *Querier) Start() {
	q.run()
}

// Stop kills the sync querier goroutine.
func (q *Querier) Stop() error {
	queryLog.Info("Stopping service")
	q.cancel()
	return nil
}

func (q *Querier) run() {
	responseSub := q.p2p.Subscribe(&pb.ChainHeadResponse{}, q.responseBuf)

	// Ticker so that service will keep on requesting for chain head
	// until they get a response.
	ticker := time.NewTicker(1 * time.Second)

	defer func() {
		responseSub.Unsubscribe()
		close(q.responseBuf)
		ticker.Stop()
	}()

	q.RequestLatestHead()

	for {
		select {
		case <-q.ctx.Done():
			queryLog.Info("Exiting goroutine")
			return
		case <-ticker.C:
			q.RequestLatestHead()
		case msg := <-q.responseBuf:
			response := msg.Data.(*pb.ChainHeadResponse)
			queryLog.Infof("Latest chain head is at slot: %d and hash %#x", response.Slot, response.Hash)
			q.curentHeadSlot = response.Slot
			q.currentHeadHash = response.Hash

			ticker.Stop()
			responseSub.Unsubscribe()
			q.cancel()
		}
	}
}

// RequestLatestHead broadcasts out a request for all
// the latest chain heads from the node's peers.
func (q *Querier) RequestLatestHead() {
	request := &pb.ChainHeadRequest{}
	q.p2p.Broadcast(request)
}

// IsSynced checks if the node is cuurently synced with the
// rest of the network.
func (q *Querier) IsSynced() (bool, error) {
	block, err := q.db.ChainHead()
	if err != nil {
		return false, err
	}

	if block.Slot >= q.curentHeadSlot {
		return true, nil
	}

	return false, err

}

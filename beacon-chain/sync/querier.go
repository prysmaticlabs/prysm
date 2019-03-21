package sync

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var queryLog = logrus.WithField("prefix", "syncQuerier")

type powChainService interface {
	HasChainStartLogOccurred() (bool, uint64, error)
	ChainStartFeed() *event.Feed
}

// QuerierConfig defines the configurable properties of SyncQuerier.
type QuerierConfig struct {
	ResponseBufferSize int
	P2P                p2pAPI
	BeaconDB           *db.BeaconDB
	PowChain           powChainService
	CurrentHeadSlot    uint64
	ChainService       chainService
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
	ctx                       context.Context
	cancel                    context.CancelFunc
	p2p                       p2pAPI
	db                        *db.BeaconDB
	chainService              chainService
	currentHeadSlot           uint64
	currentHeadHash           []byte
	currentFinalizedStateRoot [32]byte
	responseBuf               chan p2p.Message
	chainStartBuf             chan time.Time
	powchain                  powChainService
	chainStarted              bool
	atGenesis                 bool
}

// NewQuerierService constructs a new Sync Querier Service.
// This method is normally called by the main node.
func NewQuerierService(ctx context.Context,
	cfg *QuerierConfig,
) *Querier {
	ctx, cancel := context.WithCancel(ctx)

	responseBuf := make(chan p2p.Message, cfg.ResponseBufferSize)

	return &Querier{
		ctx:             ctx,
		cancel:          cancel,
		p2p:             cfg.P2P,
		db:              cfg.BeaconDB,
		chainService:    cfg.ChainService,
		responseBuf:     responseBuf,
		currentHeadSlot: cfg.CurrentHeadSlot,
		chainStarted:    false,
		powchain:        cfg.PowChain,
		chainStartBuf:   make(chan time.Time, 1),
	}
}

// Start begins the goroutine.
func (q *Querier) Start() {
	hasChainStarted, _, err := q.powchain.HasChainStartLogOccurred()
	if err != nil {
		queryLog.Errorf("Unable to get current state of the deposit contract %v", err)
		return
	}

	q.atGenesis = !hasChainStarted

	bState, err := q.db.State(q.ctx)
	if err != nil {
		queryLog.Errorf("Unable to retrieve beacon state %v", err)
	}

	// we handle both the cases where either chainstart has not occurred or
	// if beacon state has been initialized. If chain start has occurred but
	// beacon state has not been initialized we wait for the POW chain service
	// to accumulate all the deposits and process them.
	if !hasChainStarted || bState == nil {
		q.listenForStateInitialization()

		// Return, if the node is at genesis.
		if q.atGenesis {
			return
		}
	}
	q.run()
}

// Stop kills the sync querier goroutine.
func (q *Querier) Stop() error {
	queryLog.Info("Stopping service")
	q.cancel()
	return nil
}

func (q *Querier) listenForStateInitialization() {

	sub := q.chainService.StateInitializedFeed().Subscribe(q.chainStartBuf)
	defer sub.Unsubscribe()
	for {
		select {
		case <-q.chainStartBuf:
			queryLog.Info("state initialized")
			q.chainStarted = true
			return
		case <-sub.Err():
			log.Fatal("Subscriber closed, unable to continue on with sync")
			return
		case <-q.ctx.Done():
			log.Debug("RPC context closed, exiting goroutine")
			return
		}
	}
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
			q.currentHeadSlot = response.Slot
			q.currentHeadHash = response.Hash
			q.currentFinalizedStateRoot = bytesutil.ToBytes32(response.FinalizedStateRootHash32S)

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
	q.p2p.Broadcast(context.Background(), request)
}

// IsSynced checks if the node is cuurently synced with the
// rest of the network.
func (q *Querier) IsSynced() (bool, error) {
	if q.chainStarted && q.atGenesis {
		return true, nil
	}
	block, err := q.db.ChainHead()
	if err != nil {
		return false, err
	}

	if block == nil {
		return false, nil
	}

	if block.Slot >= q.currentHeadSlot {
		return true, nil
	}

	return false, err
}

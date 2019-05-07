package sync

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var queryLog = logrus.WithField("prefix", "syncQuerier")

type powChainService interface {
	HasChainStartLogOccurred() (bool, uint64, error)
	BlockExists(ctx context.Context, hash common.Hash) (bool, *big.Int, error)
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
		ResponseBufferSize: params.BeaconConfig().DefaultBufferSize,
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
	currentStateRoot          []byte
	currentFinalizedStateRoot [32]byte
	responseBuf               chan p2p.Message
	chainStartBuf             chan time.Time
	powchain                  powChainService
	chainStarted              bool
	atGenesis                 bool
	bestPeer                  peer.ID
	peerMap                   map[peer.ID]uint64
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
		atGenesis:       true,
		powchain:        cfg.PowChain,
		chainStartBuf:   make(chan time.Time, 1),
		peerMap:         make(map[peer.ID]uint64),
	}
}

// Start begins the goroutine.
func (q *Querier) Start() {
	hasChainStarted, _, err := q.powchain.HasChainStartLogOccurred()
	if err != nil {
		queryLog.Errorf("Unable to get current state of the deposit contract %v", err)
		return
	}

	q.chainStarted = hasChainStarted
	q.atGenesis = !hasChainStarted

	bState, err := q.db.HeadState(q.ctx)
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
			queryLog.Info("State has been initialized")
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

	log.Info("Polling peers for latest chain head...")
	hasReceivedResponse := false
	var timeout <-chan time.Time
	for {
		select {
		case <-q.ctx.Done():
			queryLog.Info("Finished querying state of the network, importing blocks...")
			return
		case <-ticker.C:
			q.RequestLatestHead()
		case <-timeout:
			queryLog.WithField("peerID", q.bestPeer.Pretty()).Info("Peer with highest canonical head")
			queryLog.Infof(
				"Latest chain head is at slot: %d and state root: %#x",
				q.currentHeadSlot-params.BeaconConfig().GenesisSlot, q.currentStateRoot,
			)
			ticker.Stop()
			responseSub.Unsubscribe()
			q.cancel()
		case msg := <-q.responseBuf:
			// If this is the first response a node receives, we start
			// a timeout that will keep listening for more responses over a
			// certain time interval to ensure we get the best head from our peers.
			if !hasReceivedResponse {
				timeout = time.After(10 * time.Second)
				hasReceivedResponse = true
			}
			response := msg.Data.(*pb.ChainHeadResponse)
			if _, ok := q.peerMap[msg.Peer]; !ok {
				queryLog.WithFields(logrus.Fields{
					"peerID":      msg.Peer.Pretty(),
					"highestSlot": response.CanonicalSlot - params.BeaconConfig().GenesisSlot,
				}).Info("Received chain head from peer")
				q.peerMap[msg.Peer] = response.CanonicalSlot
			}
			if response.CanonicalSlot > q.currentHeadSlot {
				q.bestPeer = msg.Peer
				q.currentHeadSlot = response.CanonicalSlot
				q.currentStateRoot = response.CanonicalStateRootHash32
				q.currentFinalizedStateRoot = bytesutil.ToBytes32(response.FinalizedStateRootHash32S)
			}
		}
	}
}

// RequestLatestHead broadcasts a request for
// the latest chain head slot and state root to a peer.
func (q *Querier) RequestLatestHead() {
	request := &pb.ChainHeadRequest{}
	q.p2p.Broadcast(context.Background(), request)
}

// IsSynced checks if the node is currently synced with the
// rest of the network.
func (q *Querier) IsSynced() (bool, error) {
	if !q.chainStarted {
		return true, nil
	}
	if q.atGenesis {
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

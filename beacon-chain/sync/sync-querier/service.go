package syncquerier

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "syncQuerier")

// Config defines the configurable properties of SyncQuerier.
//
type Config struct {
	ResponseBufferSize int
	P2P                p2pAPI
	BeaconDB           beaconDB
}

// DefaultConfig provides the default configuration for a sync service.
// ResponseBufferSize determines that buffer size of the `responseBuf` channel.
func DefaultConfig() Config {
	return Config{
		ResponseBufferSize: 100,
	}
}

type p2pAPI interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
}

type beaconDB interface {
	SaveBlock(*types.Block) error
	GetChainHead() (*types.Block, error)
}

// SyncQuerier defines the main class in this package.
// See the package comments for a general description of the service's functions.
type SyncQuerier struct {
	ctx             context.Context
	cancel          context.CancelFunc
	p2p             p2pAPI
	db              beaconDB
	curentHeadSlot  uint64
	currentHeadHash []byte
	responseBuf     chan p2p.Message
}

// NewSyncQuerierService constructs a new Sync Querier Service.
// This method is normally called by the main node.
func NewSyncQuerierService(ctx context.Context,
	cfg *Config,
) *SyncQuerier {
	ctx, cancel := context.WithCancel(ctx)

	responseBuf := make(chan p2p.Message, cfg.ResponseBufferSize)

	return &SyncQuerier{
		ctx:         ctx,
		cancel:      cancel,
		p2p:         cfg.P2P,
		db:          cfg.BeaconDB,
		responseBuf: responseBuf,
	}
}

// Start begins the goroutine.
func (s *SyncQuerier) Start() {
	s.run()
}

// Stop kills the sync querier goroutine.
func (s *SyncQuerier) Stop() error {
	log.Info("Stopping service")
	s.cancel()
	return nil
}

func (s *SyncQuerier) run() {
	responseSub := s.p2p.Subscribe(&pb.ChainHeadResponse{}, s.responseBuf)
	defer func() {
		responseSub.Unsubscribe()
		close(s.responseBuf)
	}()

	s.RequestLatestHead()

	for {
		select {
		case <-s.ctx.Done():
			log.Info("Exiting goroutine")
			return
		case msg := <-s.responseBuf:
			response := msg.Data.(*pb.ChainHeadResponse)
			log.Infof("Latest Chain head is at slot: %d and hash %#x", response.Slot, response.Hash)
			s.curentHeadSlot = response.Slot
			s.currentHeadHash = response.Hash
			responseSub.Unsubscribe()
			s.cancel()
		}
	}
}

// RequestLatestHead broadcasts out a request for all
// the latest chain heads from the node's peers.
func (s *SyncQuerier) RequestLatestHead() {
	request := &pb.ChainHeadRequest{}
	s.p2p.Broadcast(request)
}

// IsSynced checks if the node is cuurently synced with the
// rest of the network.
func (s *SyncQuerier) IsSynced() (bool, error) {
	block, err := s.db.GetChainHead()
	if err != nil {
		return false, err
	}

	if block.SlotNumber() >= s.curentHeadSlot {
		return true, nil
	}

	return false, nil
}

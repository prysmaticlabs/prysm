package syncquerier

import (
	"context"

	"github.com/golang/protobuf/proto"
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

// SyncQuerier defines the main class in this package.
// See the package comments for a general description of the service's functions.
type SyncQuerier struct {
	ctx             context.Context
	cancel          context.CancelFunc
	p2p             p2pAPI
	curentHeadSlot  uint64
	currentHeadHash uint64
	responseBuf     chan p2p.Message
}

// NewSyncQuerierService constructs a new Sync Querier Service.
// This method is normally called by the main node.
func NewSyncQuerierService(ctx context.Context,
	cfg Config,
) *SyncQuerier {
	ctx, cancel := context.WithCancel(ctx)

	responseBuf := make(chan p2p.Message, cfg.ResponseBufferSize)

	return &SyncQuerier{
		ctx:         ctx,
		cancel:      cancel,
		p2p:         cfg.P2P,
		responseBuf: responseBuf,
	}
}

// Start begins the goroutine.
func (s *SyncQuerier) Start() {
	go s.run()
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
			log.Debug("Exiting goroutine")
			return
		case msg := <-s.responseBuf:
			response := msg.Data.(*pb.ChainHeadResponse)
			log.Debugf("Latest Chain head is at slot: %d and hash %#x", response.Slot, response.Hash)

			responseSub.Unsubscribe()
		}
	}
}

func (s *SyncQuerier) RequestLatestHead() {
	request := &pb.ChainHeadRequest{}
	s.p2p.Broadcast(request)
}

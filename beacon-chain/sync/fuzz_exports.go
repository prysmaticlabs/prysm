// +build libfuzzer

package sync

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	gcache "github.com/patrickmn/go-cache"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// NewRegularSyncFuzz service without registering handlers.
func NewRegularSyncFuzz(cfg *Config) *Service {
	rLimiter := newRateLimiter(cfg.P2P)
	ctx, cancel := context.WithCancel(context.Background())
	r := &Service{
		cfg:                  cfg,
		ctx:                  ctx,
		cancel:               cancel,
		slotToPendingBlocks:  gcache.New(time.Second, 2*time.Second),
		seenPendingBlocks:    make(map[[32]byte]bool),
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		rateLimiter:          rLimiter,
	}

	return r
}

// FuzzValidateBeaconBlockPubSub exports private method validateBeaconBlockPubSub for fuzz testing.
func (s *Service) FuzzValidateBeaconBlockPubSub(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	return s.validateBeaconBlockPubSub(ctx, pid, msg)
}

// FuzzBeaconBlockSubscriber exports private method beaconBlockSubscriber for fuzz testing.
func (s *Service) FuzzBeaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	return s.beaconBlockSubscriber(ctx, msg)
}

func (s *Service) InitCaches() error {
	return s.initCaches()
}

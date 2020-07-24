// +build libfuzzer

package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// NewRegularSyncFuzz service without registering handlers.
func NewRegularSyncFuzz(cfg *Config) *Service {
	rLimiter := newRateLimiter(cfg.P2P)
	ctx, cancel := context.WithCancel(context.Background())
	r := &Service{
		ctx:                  ctx,
		cancel:               cancel,
		db:                   cfg.DB,
		p2p:                  cfg.P2P,
		attPool:              cfg.AttPool,
		exitPool:             cfg.ExitPool,
		slashingPool:         cfg.SlashingPool,
		chain:                cfg.Chain,
		initialSync:          cfg.InitialSync,
		attestationNotifier:  cfg.AttestationNotifier,
		slotToPendingBlocks:  make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:    make(map[[32]byte]bool),
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		stateNotifier:        cfg.StateNotifier,
		blockNotifier:        cfg.BlockNotifier,
		stateSummaryCache:    cfg.StateSummaryCache,
		stateGen:             cfg.StateGen,
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

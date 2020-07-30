// Package sync includes all chain-synchronization logic for the beacon node,
// including gossip-sub validators for blocks, attestations, and other p2p
// messages, as well as ability to process and respond to block requests
// by peers.
package sync

import (
	"context"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/runutil"
)

var _ = shared.Service(&Service{})

const rangeLimit = 1000
const seenBlockSize = 1000
const seenAttSize = 10000
const seenExitSize = 100
const seenAttesterSlashingSize = 100
const seenProposerSlashingSize = 100
const badBlockSize = 1000

const syncMetricsInterval = 10 * time.Second

// Config to set up the regular sync service.
type Config struct {
	P2P                 p2p.P2P
	DB                  db.NoHeadAccessDatabase
	AttPool             attestations.Pool
	ExitPool            *voluntaryexits.Pool
	SlashingPool        *slashings.Pool
	Chain               blockchainService
	InitialSync         Checker
	StateNotifier       statefeed.Notifier
	BlockNotifier       blockfeed.Notifier
	AttestationNotifier operation.Notifier
	StateSummaryCache   *cache.StateSummaryCache
	StateGen            *stategen.State
}

// This defines the interface for interacting with block chain service
type blockchainService interface {
	blockchain.BlockReceiver
	blockchain.HeadFetcher
	blockchain.FinalizationFetcher
	blockchain.ForkFetcher
	blockchain.AttestationReceiver
	blockchain.TimeFetcher
	blockchain.GenesisFetcher
	blockchain.CanonicalFetcher
}

// Service is responsible for handling all run time p2p related operations as the
// main entry point for network messages.
type Service struct {
	ctx                       context.Context
	cancel                    context.CancelFunc
	p2p                       p2p.P2P
	db                        db.NoHeadAccessDatabase
	attPool                   attestations.Pool
	exitPool                  *voluntaryexits.Pool
	slashingPool              *slashings.Pool
	chain                     blockchainService
	slotToPendingBlocks       map[uint64]*ethpb.SignedBeaconBlock
	seenPendingBlocks         map[[32]byte]bool
	blkRootToPendingAtts      map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof
	pendingAttsLock           sync.RWMutex
	pendingQueueLock          sync.RWMutex
	chainStarted              bool
	initialSync               Checker
	validateBlockLock         sync.RWMutex
	stateNotifier             statefeed.Notifier
	blockNotifier             blockfeed.Notifier
	rateLimiter               *limiter
	attestationNotifier       operation.Notifier
	seenBlockLock             sync.RWMutex
	seenBlockCache            *lru.Cache
	seenAttestationLock       sync.RWMutex
	seenAttestationCache      *lru.Cache
	seenExitLock              sync.RWMutex
	seenExitCache             *lru.Cache
	seenProposerSlashingLock  sync.RWMutex
	seenProposerSlashingCache *lru.Cache
	seenAttesterSlashingLock  sync.RWMutex
	seenAttesterSlashingCache *lru.Cache
	badBlockCache             *lru.Cache
	badBlockLock              sync.RWMutex
	stateSummaryCache         *cache.StateSummaryCache
	stateGen                  *stategen.State
}

// NewRegularSync service.
func NewRegularSync(cfg *Config) *Service {
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

	go r.registerHandlers()

	return r
}

// Start the regular sync service.
func (s *Service) Start() {
	if err := s.initCaches(); err != nil {
		panic(err)
	}

	s.p2p.AddConnectionHandler(s.reValidatePeer)
	s.p2p.AddDisconnectionHandler(func(_ context.Context, _ peer.ID) error {
		// no-op
		return nil
	})
	s.p2p.AddPingMethod(s.sendPingRequest)
	s.processPendingBlocksQueue()
	s.processPendingAttsQueue()
	s.maintainPeerStatuses()
	s.resyncIfBehind()

	// Update sync metrics.
	runutil.RunEvery(s.ctx, syncMetricsInterval, s.updateMetrics)
}

// Stop the regular sync service.
func (s *Service) Stop() error {
	defer func() {
		if s.rateLimiter != nil {
			s.rateLimiter.free()
			s.rateLimiter = nil
		}
	}()
	defer s.cancel()
	return nil
}

// Status of the currently running regular sync service.
func (s *Service) Status() error {
	if s.chainStarted {
		if s.initialSync.Syncing() {
			return errors.New("waiting for initial sync")
		}
		// If our head slot is on a previous epoch and our peers are reporting their head block are
		// in the most recent epoch, then we might be out of sync.
		if headEpoch := helpers.SlotToEpoch(s.chain.HeadSlot()); headEpoch+1 < helpers.SlotToEpoch(s.chain.CurrentSlot()) &&
			headEpoch+1 < s.p2p.Peers().HighestEpoch() {
			return errors.New("out of sync")
		}
	}
	return nil
}

// This initializes the caches to update seen beacon objects coming in from the wire
// and prevent DoS.
func (s *Service) initCaches() error {
	blkCache, err := lru.New(seenBlockSize)
	if err != nil {
		return err
	}
	attCache, err := lru.New(seenAttSize)
	if err != nil {
		return err
	}
	exitCache, err := lru.New(seenExitSize)
	if err != nil {
		return err
	}
	attesterSlashingCache, err := lru.New(seenAttesterSlashingSize)
	if err != nil {
		return err
	}
	proposerSlashingCache, err := lru.New(seenProposerSlashingSize)
	if err != nil {
		return err
	}
	badBlockCache, err := lru.New(badBlockSize)
	if err != nil {
		return err
	}
	s.seenBlockCache = blkCache
	s.seenAttestationCache = attCache
	s.seenExitCache = exitCache
	s.seenAttesterSlashingCache = attesterSlashingCache
	s.seenProposerSlashingCache = proposerSlashingCache
	s.badBlockCache = badBlockCache

	return nil
}

func (s *Service) registerHandlers() {
	// Wait until chain start.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.stateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	for s.chainStarted == false {
		select {
		case event := <-stateChannel:
			if event.Type == statefeed.Initialized {
				data, ok := event.Data.(*statefeed.InitializedData)
				if !ok {
					log.Error("Event feed data is not type *statefeed.InitializedData")
					return
				}
				log.WithField("starttime", data.StartTime).Debug("Received state initialized event")

				// Register respective rpc and pubsub handlers at state initialized event.
				s.registerRPCHandlers()
				s.registerSubscribers()

				if data.StartTime.After(roughtime.Now()) {
					stateSub.Unsubscribe()
					time.Sleep(roughtime.Until(data.StartTime))
				}
				s.chainStarted = true
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return
		case err := <-stateSub.Err():
			log.WithError(err).Error("Subscription to state notifier failed")
			return
		}
	}
}

// Checker defines a struct which can verify whether a node is currently
// synchronizing a chain with the rest of peers in the network.
type Checker interface {
	Syncing() bool
	Status() error
	Resync() error
}

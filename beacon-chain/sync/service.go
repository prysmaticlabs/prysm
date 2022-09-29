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
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	gcache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/async"
	"github.com/prysmaticlabs/prysm/v3/async/abool"
	"github.com/prysmaticlabs/prysm/v3/async/event"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/execution"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	lruwrpr "github.com/prysmaticlabs/prysm/v3/cache/lru"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime"
	prysmTime "github.com/prysmaticlabs/prysm/v3/time"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

var _ runtime.Service = (*Service)(nil)

const rangeLimit = 1024
const seenBlockSize = 1000
const seenUnaggregatedAttSize = 20000
const seenAggregatedAttSize = 1024
const seenSyncMsgSize = 1000         // Maximum of 512 sync committee members, 1000 is a safe amount.
const seenSyncContributionSize = 512 // Maximum of SYNC_COMMITTEE_SIZE as specified by the spec.
const seenExitSize = 100
const seenProposerSlashingSize = 100
const badBlockSize = 1000
const syncMetricsInterval = 10 * time.Second

var (
	// Seconds in one epoch.
	pendingBlockExpTime = time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot)) * time.Second
	// time to allow processing early blocks.
	earlyBlockProcessingTolerance = slots.MultiplySlotBy(2)
	// time to allow processing early attestations.
	earlyAttestationProcessingTolerance = params.BeaconNetworkConfig().MaximumGossipClockDisparity
	errWrongMessage                     = errors.New("wrong pubsub message")
	errNilMessage                       = errors.New("nil pubsub message")
)

// Common type for functional p2p validation options.
type validationFn func(ctx context.Context) (pubsub.ValidationResult, error)

// config to hold dependencies for the sync service.
type config struct {
	attestationNotifier           operation.Notifier
	p2p                           p2p.P2P
	beaconDB                      db.NoHeadAccessDatabase
	attPool                       attestations.Pool
	exitPool                      voluntaryexits.PoolManager
	slashingPool                  slashings.PoolManager
	syncCommsPool                 synccommittee.Pool
	chain                         blockchainService
	initialSync                   Checker
	stateNotifier                 statefeed.Notifier
	blockNotifier                 blockfeed.Notifier
	operationNotifier             operation.Notifier
	executionPayloadReconstructor execution.ExecutionPayloadReconstructor
	stateGen                      *stategen.State
	slasherAttestationsFeed       *event.Feed
	slasherBlockHeadersFeed       *event.Feed
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
	blockchain.OptimisticModeFetcher
	blockchain.SlashingReceiver
}

// Service is responsible for handling all run time p2p related operations as the
// main entry point for network messages.
type Service struct {
	cfg                              *config
	ctx                              context.Context
	cancel                           context.CancelFunc
	slotToPendingBlocks              *gcache.Cache
	seenPendingBlocks                map[[32]byte]bool
	blkRootToPendingAtts             map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof
	subHandler                       *subTopicHandler
	pendingAttsLock                  sync.RWMutex
	pendingQueueLock                 sync.RWMutex
	chainStarted                     *abool.AtomicBool
	validateBlockLock                sync.RWMutex
	rateLimiter                      *limiter
	seenBlockLock                    sync.RWMutex
	seenBlockCache                   *lru.Cache
	seenAggregatedAttestationLock    sync.RWMutex
	seenAggregatedAttestationCache   *lru.Cache
	seenUnAggregatedAttestationLock  sync.RWMutex
	seenUnAggregatedAttestationCache *lru.Cache
	seenExitLock                     sync.RWMutex
	seenExitCache                    *lru.Cache
	seenProposerSlashingLock         sync.RWMutex
	seenProposerSlashingCache        *lru.Cache
	seenAttesterSlashingLock         sync.RWMutex
	seenAttesterSlashingCache        map[uint64]bool
	seenSyncMessageLock              sync.RWMutex
	seenSyncMessageCache             *lru.Cache
	seenSyncContributionLock         sync.RWMutex
	seenSyncContributionCache        *lru.Cache
	badBlockCache                    *lru.Cache
	badBlockLock                     sync.RWMutex
	syncContributionBitsOverlapLock  sync.RWMutex
	syncContributionBitsOverlapCache *lru.Cache
	signatureChan                    chan *signatureVerifier
}

// NewService initializes new regular sync service.
func NewService(ctx context.Context, opts ...Option) *Service {
	c := gcache.New(pendingBlockExpTime /* exp time */, 2*pendingBlockExpTime /* prune time */)
	ctx, cancel := context.WithCancel(ctx)
	r := &Service{
		ctx:                  ctx,
		cancel:               cancel,
		chainStarted:         abool.New(),
		cfg:                  &config{},
		slotToPendingBlocks:  c,
		seenPendingBlocks:    make(map[[32]byte]bool),
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		signatureChan:        make(chan *signatureVerifier, verifierLimit),
	}
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil
		}
	}
	r.subHandler = newSubTopicHandler()
	r.rateLimiter = newRateLimiter(r.cfg.p2p)
	r.initCaches()

	go r.registerHandlers()
	go r.verifierRoutine()

	return r
}

// Start the regular sync service.
func (s *Service) Start() {
	s.cfg.p2p.AddConnectionHandler(s.reValidatePeer, s.sendGoodbye)
	s.cfg.p2p.AddDisconnectionHandler(func(_ context.Context, _ peer.ID) error {
		// no-op
		return nil
	})
	s.cfg.p2p.AddPingMethod(s.sendPingRequest)
	s.processPendingBlocksQueue()
	s.processPendingAttsQueue()
	s.maintainPeerStatuses()
	s.resyncIfBehind()

	// Update sync metrics.
	async.RunEvery(s.ctx, syncMetricsInterval, s.updateMetrics)
}

// Stop the regular sync service.
func (s *Service) Stop() error {
	defer func() {
		if s.rateLimiter != nil {
			s.rateLimiter.free()
		}
	}()
	// Removing RPC Stream handlers.
	for _, p := range s.cfg.p2p.Host().Mux().Protocols() {
		s.cfg.p2p.Host().RemoveStreamHandler(protocol.ID(p))
	}
	// Deregister Topic Subscribers.
	for _, t := range s.cfg.p2p.PubSub().GetTopics() {
		s.unSubscribeFromTopic(t)
	}
	defer s.cancel()
	return nil
}

// Status of the currently running regular sync service.
func (s *Service) Status() error {
	// If our head slot is on a previous epoch and our peers are reporting their head block are
	// in the most recent epoch, then we might be out of sync.
	if headEpoch := slots.ToEpoch(s.cfg.chain.HeadSlot()); headEpoch+1 < slots.ToEpoch(s.cfg.chain.CurrentSlot()) &&
		headEpoch+1 < s.cfg.p2p.Peers().HighestEpoch() {
		return errors.New("out of sync")
	}
	return nil
}

// This initializes the caches to update seen beacon objects coming in from the wire
// and prevent DoS.
func (s *Service) initCaches() {
	s.seenBlockCache = lruwrpr.New(seenBlockSize)
	s.seenAggregatedAttestationCache = lruwrpr.New(seenAggregatedAttSize)
	s.seenUnAggregatedAttestationCache = lruwrpr.New(seenUnaggregatedAttSize)
	s.seenSyncMessageCache = lruwrpr.New(seenSyncMsgSize)
	s.seenSyncContributionCache = lruwrpr.New(seenSyncContributionSize)
	s.syncContributionBitsOverlapCache = lruwrpr.New(seenSyncContributionSize)
	s.seenExitCache = lruwrpr.New(seenExitSize)
	s.seenAttesterSlashingCache = make(map[uint64]bool)
	s.seenProposerSlashingCache = lruwrpr.New(seenProposerSlashingSize)
	s.badBlockCache = lruwrpr.New(badBlockSize)
}

func (s *Service) registerHandlers() {
	// Wait until chain start.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.cfg.stateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	for {
		select {
		case e := <-stateChannel:
			switch e.Type {
			case statefeed.Initialized:
				data, ok := e.Data.(*statefeed.InitializedData)
				if !ok {
					log.Error("Event feed data is not type *statefeed.InitializedData")
					return
				}
				startTime := data.StartTime
				log.WithField("starttime", startTime).Debug("Received state initialized event")

				// Register respective rpc handlers at state initialized event.
				s.registerRPCHandlers()
				// Wait for chainstart in separate routine.
				go func() {
					if startTime.After(prysmTime.Now()) {
						time.Sleep(prysmTime.Until(startTime))
					}
					log.WithField("starttime", startTime).Debug("Chain started in sync service")
					s.markForChainStart()
				}()
			case statefeed.Synced:
				_, ok := e.Data.(*statefeed.SyncedData)
				if !ok {
					log.Error("Event feed data is not type *statefeed.SyncedData")
					return
				}
				// Register respective pubsub handlers at state synced event.
				digest, err := s.currentForkDigest()
				if err != nil {
					log.WithError(err).Error("Could not retrieve current fork digest")
					return
				}
				currentEpoch := slots.ToEpoch(slots.CurrentSlot(uint64(s.cfg.chain.GenesisTime().Unix())))
				s.registerSubscribers(currentEpoch, digest)
				go s.forkWatcher()
				return
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return
		case err := <-stateSub.Err():
			log.WithError(err).Error("Could not subscribe to state notifier")
			return
		}
	}
}

// marks the chain as having started.
func (s *Service) markForChainStart() {
	s.chainStarted.Set()
}

// Checker defines a struct which can verify whether a node is currently
// synchronizing a chain with the rest of peers in the network.
type Checker interface {
	Initialized() bool
	Syncing() bool
	Synced() bool
	Status() error
	Resync() error
}

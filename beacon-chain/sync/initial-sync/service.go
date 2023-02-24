// Package initialsync includes all initial block download and processing
// logic for the beacon node, using a round robin strategy and a finite-state-machine
// to handle edge-cases in a beacon node's sync status.
package initialsync

import (
	"context"
	"time"

	"github.com/paulbellamy/ratecounter"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/async/abool"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v3/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/runtime"
	prysmTime "github.com/prysmaticlabs/prysm/v3/time"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
)

var _ runtime.Service = (*Service)(nil)

// blockchainService defines the interface for interaction with block chain service.
type blockchainService interface {
	blockchain.BlockReceiver
	blockchain.ChainInfoFetcher
}

// Config to set up the initial sync service.
type Config struct {
	P2P           p2p.P2P
	DB            db.ReadOnlyDatabase
	Chain         blockchainService
	StateNotifier statefeed.Notifier
	BlockNotifier blockfeed.Notifier
}

// Service service.
type Service struct {
	cfg          *Config
	ctx          context.Context
	cancel       context.CancelFunc
	synced       *abool.AtomicBool
	chainStarted *abool.AtomicBool
	counter      *ratecounter.RateCounter
	genesisChan  chan time.Time
}

// NewService configures the initial sync service responsible for bringing the node up to the
// latest head of the blockchain.
func NewService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	s := &Service{
		cfg:          cfg,
		ctx:          ctx,
		cancel:       cancel,
		synced:       abool.New(),
		chainStarted: abool.New(),
		counter:      ratecounter.NewRateCounter(counterSeconds * time.Second),
		genesisChan:  make(chan time.Time),
	}

	// The reason why we have this goroutine in the constructor is to avoid a race condition
	// between services' Start method and the initialization event.
	// See https://github.com/prysmaticlabs/prysm/issues/10602 for details.
	go s.waitForStateInitialization()

	return s
}

// Start the initial sync service.
func (s *Service) Start() {
	// Wait for state initialized event.
	genesis := <-s.genesisChan
	if genesis.IsZero() {
		log.Debug("Exiting Initial Sync Service")
		return
	}
	if genesis.After(prysmTime.Now()) {
		s.markSynced(genesis)
		log.WithField("genesisTime", genesis).Info("Genesis time has not arrived - not syncing")
		return
	}
	currentSlot := slots.Since(genesis)
	if slots.ToEpoch(currentSlot) == 0 {
		log.WithField("genesisTime", genesis).Info("Chain started within the last epoch - not syncing")
		s.markSynced(genesis)
		return
	}
	s.chainStarted.Set()
	log.Info("Starting initial chain sync...")
	// Are we already in sync, or close to it?
	if slots.ToEpoch(s.cfg.Chain.HeadSlot()) == slots.ToEpoch(currentSlot) {
		log.Info("Already synced to the current chain head")
		s.markSynced(genesis)
		return
	}
	s.waitForMinimumPeers()
	if err := s.roundRobinSync(genesis); err != nil {
		if errors.Is(s.ctx.Err(), context.Canceled) {
			return
		}
		panic(err)
	}
	log.Infof("Synced up to slot %d", s.cfg.Chain.HeadSlot())
	s.markSynced(genesis)
}

// Stop initial sync.
func (s *Service) Stop() error {
	s.cancel()
	return nil
}

// Status of initial sync.
func (s *Service) Status() error {
	if s.synced.IsNotSet() && s.chainStarted.IsSet() {
		return errors.New("syncing")
	}
	return nil
}

// Syncing returns true if initial sync is still running.
func (s *Service) Syncing() bool {
	return s.synced.IsNotSet()
}

// Initialized returns true if initial sync has been started.
func (s *Service) Initialized() bool {
	return s.chainStarted.IsSet()
}

// Synced returns true if initial sync has been completed.
func (s *Service) Synced() bool {
	return s.synced.IsSet()
}

// Resync allows a node to start syncing again if it has fallen
// behind the current network head.
func (s *Service) Resync() error {
	headState, err := s.cfg.Chain.HeadState(s.ctx)
	if err != nil || headState == nil || headState.IsNil() {
		return errors.Errorf("could not retrieve head state: %v", err)
	}

	// Set it to false since we are syncing again.
	s.synced.UnSet()
	defer func() { s.synced.Set() }()                       // Reset it at the end of the method.
	genesis := time.Unix(int64(headState.GenesisTime()), 0) // lint:ignore uintcast -- Genesis time will not exceed int64 in your lifetime.

	s.waitForMinimumPeers()
	if err = s.roundRobinSync(genesis); err != nil {
		log = log.WithError(err)
	}
	log.WithField("slot", s.cfg.Chain.HeadSlot()).Info("Resync attempt complete")
	return nil
}

func (s *Service) waitForMinimumPeers() {
	required := params.BeaconConfig().MaxPeersToSync
	if flags.Get().MinimumSyncPeers < required {
		required = flags.Get().MinimumSyncPeers
	}
	for {
		cp := s.cfg.Chain.FinalizedCheckpt()
		_, peers := s.cfg.P2P.Peers().BestNonFinalized(flags.Get().MinimumSyncPeers, cp.Epoch)
		if len(peers) >= required {
			break
		}
		log.WithFields(logrus.Fields{
			"suitable": len(peers),
			"required": required,
		}).Info("Waiting for enough suitable peers before syncing")
		time.Sleep(handshakePollingInterval)
	}
}

// waitForStateInitialization makes sure that beacon node is ready to be accessed: it is either
// already properly configured or system waits up until state initialized event is triggered.
func (s *Service) waitForStateInitialization() {
	// Wait for state to be initialized.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.cfg.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	log.Info("Waiting for state to be initialized")
	for {
		select {
		case event := <-stateChannel:
			if event.Type == statefeed.Initialized {
				data, ok := event.Data.(*statefeed.InitializedData)
				if !ok {
					log.Error("Event feed data is not type *statefeed.InitializedData")
					continue
				}
				log.WithField("starttime", data.StartTime).Debug("Received state initialized event")
				s.genesisChan <- data.StartTime
				return
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			// Send a zero time in the event we are exiting.
			s.genesisChan <- time.Time{}
			return
		case err := <-stateSub.Err():
			log.WithError(err).Error("Subscription to state notifier failed")
			// Send a zero time in the event we are exiting.
			s.genesisChan <- time.Time{}
			return
		}
	}
}

// markSynced marks node as synced and notifies feed listeners.
func (s *Service) markSynced(genesis time.Time) {
	s.synced.Set()
	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Synced,
		Data: &statefeed.SyncedData{
			StartTime: genesis,
		},
	})
}

// Package light implements necessary components to support light clients.
package light

import (
	"context"
	"sync"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/container/queue"
)

// Config for the light service in the beacon node.
// This struct allows us to specify required dependencies and
// parameters to support light client to function as needed.
type Config struct {
	BeaconDB      db.NoHeadAccessDatabase
	StateNotifier statefeed.Notifier
}

// Service defining a light server implementation as part of
// the beacon node.
type Service struct {
	cfg             *Config
	ctx             context.Context
	cancel          context.CancelFunc
	updateCache     *queue.PriorityQueue
	updateCacheLock sync.RWMutex
	genesisTime time.Time
}

// New instantiates a new light service from configuration values.
func New(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:         ctx,
		cancel:      cancel,
		cfg:         cfg,
		updateCache: queue.New(),
	}
}

// Start the light service.
func (s *Service) Start() {
	go s.run() // Start functions must be non-blocking.
}

// Stop the light service.
func (s *Service) Stop() error {
	s.cancel()
	return nil
}

// Status of the slasher service.
func (_ *Service) Status() error {
	return nil
}

func (s *Service) run() {
	s.waitForChainInitialization()
	go s.subscribeEvents(s.ctx)
}

func (s *Service) waitForChainInitialization() {
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.cfg.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	defer close(stateChannel)
	for {
		select {
		case stateEvent := <-stateChannel:
			if stateEvent.Type == statefeed.Initialized {
				data, ok := stateEvent.Data.(*statefeed.InitializedData)
				if !ok {
					log.Error("Could not receive chain start notification, want *statefeed.ChainStartedData")
					return
				}
				s.genesisTime = data.StartTime
				log.WithField("genesisTime", data.StartTime).Info("Light service received chain initialization event")
				return
			}
		case err := <-stateSub.Err():
			log.WithError(err).Error(
				"Slasher could not subscribe to state events",
			)
			return
		case <-s.ctx.Done():
			return
		}
	}
}


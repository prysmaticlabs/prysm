// Package slasher --
package slasher

import (
	"context"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
)

// ServiceConfig for the slasher service in the beacon node.
// This struct allows us to specify required dependencies and
// parameters for slasher to function as needed.
type ServiceConfig struct {
	IndexedAttestationsFeed *event.Feed
	BeaconBlockHeadersFeed  *event.Feed
	AttesterSlashingsFeed   *event.Feed
	ProposerSlashingsFeed   *event.Feed
	Database                db.Database
	StateNotifier           statefeed.Notifier
}

// Service defining a slasher implementation as part of
// the beacon node, able to detect eth2 slashable offenses.
type Service struct {
	params                 *Parameters
	serviceCfg             *ServiceConfig
	indexedAttsChan        chan *ethpb.IndexedAttestation
	beaconBlockHeadersChan chan *ethpb.SignedBeaconBlockHeader
	attsQueue              *attestationsQueue
	blksQueue              *blocksQueue
	ctx                    context.Context
	cancel                 context.CancelFunc
	slotTicker             *slotutil.SlotTicker
}

// New instantiates a new slasher from configuration values.
func New(ctx context.Context, srvCfg *ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		params:                 DefaultParams(),
		serviceCfg:             srvCfg,
		indexedAttsChan:        make(chan *ethpb.IndexedAttestation, 1),
		beaconBlockHeadersChan: make(chan *ethpb.SignedBeaconBlockHeader, 1),
		attsQueue:              newAttestationsQueue(),
		blksQueue:              newBlocksQueue(),
		ctx:                    ctx,
		cancel:                 cancel,
	}, nil
}

// Start listening for received indexed attestations and blocks
// and perform slashing detection on them.
func (s *Service) Start() {
	var genesisTime time.Time
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.serviceCfg.StateNotifier.StateFeed().Subscribe(stateChannel)
	event := <-stateChannel

	// Wait for us to receive the genesis time via a chain started notification.
	if event.Type == statefeed.ChainStarted {
		data, ok := event.Data.(*statefeed.ChainStartedData)
		if !ok {
			log.Error("Could not receive chain start notification, want *statefeed.ChainStartedData")
			return
		}
		genesisTime = data.StartTime
		log.WithField("genesisTime", genesisTime).Info("Starting slasher, received chain start event")
	} else if event.Type == statefeed.Initialized {
		// Alternatively, if the chain has already started, we then read the genesis
		// time value from this data.
		data, ok := event.Data.(*statefeed.InitializedData)
		if !ok {
			log.Error("Could not receive chain start notification, want *statefeed.ChainStartedData")
			return
		}
		genesisTime = data.StartTime
		log.WithField("genesisTime", genesisTime).Info("Starting slasher, chain already initialized")
	} else {
		// This should not happen.
		log.Error("Could start slasher, could not receive chain start event")
		return
	}
	stateSub.Unsubscribe()
	secondsPerSlot := params.BeaconConfig().SecondsPerSlot
	s.slotTicker = slotutil.NewSlotTicker(genesisTime, secondsPerSlot)

	go s.processQueuedAttestations(s.ctx, s.slotTicker.C())
	go s.processQueuedBlocks(s.ctx, s.slotTicker.C())
	go s.receiveAttestations(s.ctx)
	go s.receiveBlocks(s.ctx)
	go s.pruneSlasherData(s.ctx, s.slotTicker.C())
}

// Stop the slasher service.
func (s *Service) Stop() error {
	s.cancel()
	if s.slotTicker != nil {
		s.slotTicker.Done()
	}
	return nil
}

// Status of the slasher service.
func (s *Service) Status() error {
	return nil
}

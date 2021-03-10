// Package slasher --
package slasher

import (
	"context"
	"sync"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
)

// ServiceConfig for the slasher service in the beacon node.
// This struct allows us to specify required dependencies and
// parameters for slasher to function as needed.
type ServiceConfig struct {
	IndexedAttsFeed       *event.Feed
	BeaconBlocksFeed      *event.Feed
	AttesterSlashingsFeed *event.Feed
	ProposerSlashingsFeed *event.Feed
	Database              db.Database
	GenesisTime           time.Time
}

// Service defining a slasher implementation as part of
// the beacon node, able to detect eth2 slashable offenses.
type Service struct {
	params               *Parameters
	serviceCfg           *ServiceConfig
	indexedAttsChan      chan *ethpb.IndexedAttestation
	beaconBlocksChan     chan *ethpb.SignedBeaconBlockHeader
	attestationQueueLock sync.Mutex
	blockQueueLock       sync.Mutex
	attestationQueue     []*slashertypes.IndexedAttestationWrapper
	beaconBlocksQueue    []*slashertypes.SignedBlockHeaderWrapper
	ctx                  context.Context
	cancel               context.CancelFunc
	genesisTime          time.Time
	slotTicker           slotutil.Ticker
}

// New instantiates a new slasher from configuration values.
func New(ctx context.Context, srvCfg *ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		params:            DefaultParams(),
		serviceCfg:        srvCfg,
		indexedAttsChan:   make(chan *ethpb.IndexedAttestation, 1),
		beaconBlocksChan:  make(chan *ethpb.SignedBeaconBlockHeader, 1),
		attestationQueue:  make([]*slashertypes.IndexedAttestationWrapper, 0),
		beaconBlocksQueue: make([]*slashertypes.SignedBlockHeaderWrapper, 0),
		ctx:               ctx,
		cancel:            cancel,
		genesisTime:       time.Now(),
	}, nil
}

// Start listening for received indexed attestations and blocks
// and perform slashing detection on them.
func (s *Service) Start() {
	log.Info("Starting slasher")
	s.slotTicker = slotutil.NewSlotTicker(s.genesisTime, params.BeaconConfig().SecondsPerSlot)
	go s.processQueuedAttestations(s.ctx, s.slotTicker.C())
	go s.processQueuedBlocks(s.ctx, s.slotTicker.C())
	go s.receiveBlocks(s.ctx)
	go s.receiveAttestations(s.ctx)
}

// Stop the slasher service.
func (s *Service) Stop() error {
	s.cancel()
	s.slotTicker.Done()
	return nil
}

// Status of the slasher service.
func (s *Service) Status() error {
	return nil
}

// Package slasher --
package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
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
	GenesisTimeFetcher    blockchain.TimeFetcher
}

// Service defining a slasher implementation as part of
// the beacon node, able to detect eth2 slashable offenses.
type Service struct {
	params           *Parameters
	serviceCfg       *ServiceConfig
	indexedAttsChan  chan *ethpb.IndexedAttestation
	beaconBlocksChan chan *ethpb.SignedBeaconBlockHeader
	attsQueue        *attestationsQueue
	blksQueue        *blocksQueue
	ctx              context.Context
	cancel           context.CancelFunc
	slotTicker       slotutil.Ticker
}

// New instantiates a new slasher from configuration values.
func New(ctx context.Context, srvCfg *ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		params:           DefaultParams(),
		serviceCfg:       srvCfg,
		indexedAttsChan:  make(chan *ethpb.IndexedAttestation, 1),
		beaconBlocksChan: make(chan *ethpb.SignedBeaconBlockHeader, 1),
		attsQueue:        newAttestationsQueue(),
		blksQueue:        newBlocksQueue(),
		ctx:              ctx,
		cancel:           cancel,
	}, nil
}

// Start listening for received indexed attestations and blocks
// and perform slashing detection on them.
func (s *Service) Start() {
	genesisTime := s.serviceCfg.GenesisTimeFetcher.GenesisTime()
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
	s.slotTicker.Done()
	return nil
}

// Status of the slasher service.
func (s *Service) Status() error {
	return nil
}

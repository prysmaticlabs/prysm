package simulator

import (
	"context"
	"sync"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/slasher"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/sirupsen/logrus"
)

// Parameters for a slasher simulator.
type Parameters struct {
	SecondsPerSlot         uint64
	AggregationPercent     float64
	ProposerSlashingProbab float64
	AttesterSlashingProbab float64
	NumValidators          uint64
	NumEpochs              uint64
}

// Simulator defines a struct which can launch a slasher simulation
// at scale using configuration parameters.
type Simulator struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	slasher               *slasher.Service
	params                *Parameters
	indexedAttsFeed       *event.Feed
	beaconBlocksFeed      *event.Feed
	blockSlashingFeed     *event.Feed
	attSlasingFeed        *event.Feed
	sentAttSlashingFeed   *event.Feed
	sentBlockSlashingFeed *event.Feed
	detectedSlashings     map[[32]byte]bool
	sentSlashings         map[[32]byte]bool
	genesisTime           time.Time
	lock                  sync.RWMutex
}

// DefaultParams for launching a slasher simulator.
func DefaultParams() *Parameters {
	return &Parameters{
		SecondsPerSlot:         1,
		AggregationPercent:     1.0,
		ProposerSlashingProbab: 0.2,
		AttesterSlashingProbab: 0.2,
		NumValidators:          32,
		NumEpochs:              2,
	}
}

// New initializes a slasher simulator from a beacon database
// and configuration parameters.
func New(ctx context.Context, beaconDB db.Database) (*Simulator, error) {
	indexedAttsFeed := new(event.Feed)
	beaconBlocksFeed := new(event.Feed)
	attSlashingFeed := new(event.Feed)
	sentBlockSlashingFeed := new(event.Feed)
	sentAttSlashingFeed := new(event.Feed)
	blockSlashingFeed := new(event.Feed)
	sentSlashings := make(map[[32]byte]bool)
	detectedSlashings := make(map[[32]byte]bool)
	genesisTime := time.Now()
	slasherSrv, err := slasher.New(ctx, &slasher.ServiceConfig{
		IndexedAttsFeed:    indexedAttsFeed,
		BeaconBlocksFeed:   beaconBlocksFeed,
		AttSlashingsFeed:   attSlashingFeed,
		BlockSlashingsFeed: blockSlashingFeed,
		Database:           beaconDB,
		GenesisTime:        genesisTime,
	})
	if err != nil {
		return nil, err
	}
	return &Simulator{
		ctx:                   ctx,
		slasher:               slasherSrv,
		params:                DefaultParams(),
		indexedAttsFeed:       indexedAttsFeed,
		beaconBlocksFeed:      beaconBlocksFeed,
		attSlasingFeed:        attSlashingFeed,
		blockSlashingFeed:     blockSlashingFeed,
		sentAttSlashingFeed:   sentAttSlashingFeed,
		sentBlockSlashingFeed: sentBlockSlashingFeed,
		sentSlashings:         sentSlashings,
		detectedSlashings:     detectedSlashings,
		genesisTime:           genesisTime,
	}, nil
}

// Start a simulator.
func (s *Simulator) Start() {
	log.WithFields(logrus.Fields{
		"numValidators":          s.params.NumValidators,
		"numEpochs":              s.params.NumEpochs,
		"secondsPerSlot":         s.params.SecondsPerSlot,
		"proposerSlashingProbab": s.params.ProposerSlashingProbab,
		"attesterSlashingProbab": s.params.AttesterSlashingProbab,
	}).Info("Starting slasher simulator")

	// Override global configuration for simulation purposes.
	config := params.BeaconConfig().Copy()
	config.SecondsPerSlot = s.params.SecondsPerSlot
	params.OverrideBeaconConfig(config)
	defer params.OverrideBeaconConfig(params.BeaconConfig())

	// Start slasher in the background (Start() is non-blocking).
	s.slasher.Start()

	// We simulate blocks and attestations for N epochs, and in the background,
	// start a routine which collects slashings detected by the running slasher.
	go s.receiveDetectedSlashings(s.ctx)
	s.simulateBlocksAndAttestations(s.ctx)

	// Verify the slashings we detected are the same as those the
	// simulator produced, effectively checking slasher caught all slashable offenses.
	s.verifySlashingsWereDetected(s.ctx)
	return
}

// Stop the simulator.
func (s *Simulator) Stop() error {
	return s.slasher.Stop()
}

func (s *Simulator) simulateBlocksAndAttestations(ctx context.Context) {
	ticker := slotutil.NewSlotTicker(s.genesisTime.Add(time.Millisecond*500), params.BeaconConfig().SecondsPerSlot)
	defer ticker.Done()
	for {
		select {
		case slot := <-ticker.C():
			// We only run the simulator for a specified number of epochs.
			if helpers.SlotToEpoch(slot) >= types.Epoch(s.params.NumEpochs) {
				return
			}

			blockHeaders, propSlashings := generateBlockHeadersForSlot(s.params, slot)
			log.WithFields(logrus.Fields{
				"numBlocks":    len(blockHeaders),
				"numSlashable": len(propSlashings),
			}).Infof("Producing blocks for slot %d", slot)
			// TODO: Some logic here is duplicated, we can use some abstraction here.
			for _, sl := range propSlashings {
				slashingRoot, err := sl.HashTreeRoot()
				if err != nil {
					log.WithError(err).Fatal("Could not hash tree root slashing")
				}
				s.sentSlashings[slashingRoot] = true
			}
			for _, bb := range blockHeaders {
				s.beaconBlocksFeed.Send(bb)
			}
			atts, attSlashings := generateAttestationsForSlot(s.params, slot)
			log.WithFields(logrus.Fields{
				"numAtts":      len(atts),
				"numSlashable": len(propSlashings),
			}).Infof("Producing attestations for slot %d", slot)
			for _, sl := range attSlashings {
				slashingRoot, err := sl.HashTreeRoot()
				if err != nil {
					log.WithError(err).Fatal("Could not hash tree root slashing")
				}
				s.sentSlashings[slashingRoot] = true
			}
			for _, aa := range atts {
				s.indexedAttsFeed.Send(aa)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Simulator) receiveDetectedSlashings(ctx context.Context) {
	detectedSlashings := make(chan *feed.Event, 1)
	attSub := s.attSlasingFeed.Subscribe(detectedSlashings)
	blockSub := s.blockSlashingFeed.Subscribe(detectedSlashings)
	defer func() {
		attSub.Unsubscribe()
		attSub.Unsubscribe()
	}()
	for {
		select {
		case detectedEvent := <-detectedSlashings:
			var slashingRoot [32]byte
			var err error
			switch detectedEvent.Type {
			case slashertypes.AttesterSlashing:
				attSlashing, ok := detectedEvent.Data.(*ethpb.AttesterSlashing)
				if !ok {
					log.Fatal("Detected slashing is not of type AttesterSlashing")
				}
				slashingRoot, err = attSlashing.HashTreeRoot()
				if err != nil {
					log.WithError(err).Fatal("Could not hash tree root attester slashing")
				}
			case slashertypes.ProposerSlashing:
				proposerSlashing, ok := detectedEvent.Data.(*ethpb.ProposerSlashing)
				if !ok {
					log.Fatal("Detected slashing is not of type ProposerSlashing")
				}
				slashingRoot, err = proposerSlashing.HashTreeRoot()
				if err != nil {
					log.WithError(err).Fatal("Could not hash tree root attester slashing")
				}
			}
			s.lock.Lock()
			s.detectedSlashings[slashingRoot] = true
			s.lock.Unlock()
		case <-ctx.Done():
			return
		case err := <-attSub.Err():
			log.WithError(err).Fatal("Error from attester slashing feed subscription")
			return
		case err := <-blockSub.Err():
			log.WithError(err).Fatal("Error from attester slashing feed subscription")
			return
		}
	}
}

func (s *Simulator) verifySlashingsWereDetected(ctx context.Context) {
	for slashingRoot := range s.sentSlashings {
		_, ok := s.detectedSlashings[slashingRoot]
		if ok {
			log.Infof("Correctly detected simulated slashing with root %#x", slashingRoot)
		} else {
			log.Errorf("Did not detect simulated slashing with root %#x", slashingRoot)
		}
	}
}

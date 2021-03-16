package simulator

import (
	"context"
	"sync"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/slasher"
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
	ctx                       context.Context
	slasher                   *slasher.Service
	params                    *Parameters
	indexedAttsFeed           *event.Feed
	beaconBlocksFeed          *event.Feed
	attesterSlashingsFeed     *event.Feed
	proposerSlashingsFeed     *event.Feed
	sentAttSlashingFeed       *event.Feed
	sentBlockSlashingFeed     *event.Feed
	detectedProposerSlashings map[[32]byte]*ethpb.ProposerSlashing
	detectedAttesterSlashings map[[32]byte]*ethpb.AttesterSlashing
	sentProposerSlashings     map[[32]byte]*ethpb.ProposerSlashing
	sentAttesterSlashings     map[[32]byte]*ethpb.AttesterSlashing
	genesisTime               time.Time
	proposerSlashingLock      sync.RWMutex
	attesterSlashingLock      sync.RWMutex
}

// DefaultParams for launching a slasher simulator.
func DefaultParams() *Parameters {
	return &Parameters{
		SecondsPerSlot:         2,
		AggregationPercent:     1.0,
		ProposerSlashingProbab: 0.2,
		AttesterSlashingProbab: 0.2,
		NumValidators:          128,
		NumEpochs:              2,
	}
}

// New initializes a slasher simulator from a beacon database
// and configuration parameters.
func New(ctx context.Context, beaconDB db.Database) (*Simulator, error) {
	indexedAttsFeed := new(event.Feed)
	beaconBlocksFeed := new(event.Feed)
	sentBlockSlashingFeed := new(event.Feed)
	sentAttSlashingFeed := new(event.Feed)
	attesterSlashingsFeed := new(event.Feed)
	proposerSlashingsFeed := new(event.Feed)

	slasherSrv, err := slasher.New(ctx, &slasher.ServiceConfig{
		IndexedAttestationsFeed: indexedAttsFeed,
		BeaconBlockHeadersFeed:  beaconBlocksFeed,
		AttesterSlashingsFeed:   attesterSlashingsFeed,
		ProposerSlashingsFeed:   proposerSlashingsFeed,
		Database:                beaconDB,
		StateNotifier:           &mock.MockStateNotifier{},
	})
	if err != nil {
		return nil, err
	}
	return &Simulator{
		ctx:                       ctx,
		slasher:                   slasherSrv,
		params:                    DefaultParams(),
		indexedAttsFeed:           indexedAttsFeed,
		beaconBlocksFeed:          beaconBlocksFeed,
		attesterSlashingsFeed:     attesterSlashingsFeed,
		proposerSlashingsFeed:     proposerSlashingsFeed,
		sentAttSlashingFeed:       sentAttSlashingFeed,
		sentBlockSlashingFeed:     sentBlockSlashingFeed,
		sentProposerSlashings:     make(map[[32]byte]*ethpb.ProposerSlashing),
		detectedProposerSlashings: make(map[[32]byte]*ethpb.ProposerSlashing),
		sentAttesterSlashings:     make(map[[32]byte]*ethpb.AttesterSlashing),
		detectedAttesterSlashings: make(map[[32]byte]*ethpb.AttesterSlashing),
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
	go s.receiveDetectedAttesterSlashings(s.ctx)
	go s.receiveDetectedProposerSlashings(s.ctx)
	s.simulateBlocksAndAttestations(s.ctx)

	// Verify the slashings we detected are the same as those the
	// simulator produced, effectively checking slasher caught all slashable offenses.
	s.verifySlashingsWereDetected(s.ctx)
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
			if helpers.SlotToEpoch(slot)+1 >= types.Epoch(s.params.NumEpochs) {
				return
			}

			blockHeaders, propSlashings := generateBlockHeadersForSlot(s.params, slot)
			log.WithFields(logrus.Fields{
				"numBlocks":    len(blockHeaders),
				"numSlashable": len(propSlashings),
			}).Infof("Producing blocks for slot %d", slot)
			for _, sl := range propSlashings {
				slashingRoot, err := sl.HashTreeRoot()
				if err != nil {
					log.WithError(err).Fatal("Could not hash tree root slashing")
				}
				s.sentProposerSlashings[slashingRoot] = sl
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
				s.sentAttesterSlashings[slashingRoot] = sl
			}
			for _, aa := range atts {
				s.indexedAttsFeed.Send(aa)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Simulator) receiveDetectedProposerSlashings(ctx context.Context) {
	proposerSlashingChan := make(chan *ethpb.ProposerSlashing, 1)
	sub := s.proposerSlashingsFeed.Subscribe(proposerSlashingChan)
	defer sub.Unsubscribe()
	for {
		select {
		case slashing := <-proposerSlashingChan:
			slashingRoot, err := slashing.HashTreeRoot()
			if err != nil {
				log.WithError(err).Fatal("Could not hash tree root proposer slashing")
			}
			s.proposerSlashingLock.Lock()
			s.detectedProposerSlashings[slashingRoot] = slashing
			s.proposerSlashingLock.Unlock()
		case <-ctx.Done():
			return
		case err := <-sub.Err():
			log.WithError(err).Fatal("Error from attester slashing feed subscription")
			return
		}
	}
}

func (s *Simulator) receiveDetectedAttesterSlashings(ctx context.Context) {
	attesterSlashingChan := make(chan *ethpb.AttesterSlashing, 1)
	sub := s.attesterSlashingsFeed.Subscribe(attesterSlashingChan)
	defer sub.Unsubscribe()
	for {
		select {
		case slashing := <-attesterSlashingChan:
			slashingRoot, err := slashing.HashTreeRoot()
			if err != nil {
				log.WithError(err).Fatal("Could not hash tree root attester slashing")
			}
			s.attesterSlashingLock.Lock()
			s.detectedAttesterSlashings[slashingRoot] = slashing
			s.attesterSlashingLock.Unlock()
		case <-ctx.Done():
			return
		case err := <-sub.Err():
			log.WithError(err).Fatal("Error from attester slashing feed subscription")
			return
		}
	}
}

func (s *Simulator) verifySlashingsWereDetected(ctx context.Context) {
	for slashingRoot, slashing := range s.sentProposerSlashings {
		if _, ok := s.detectedProposerSlashings[slashingRoot]; !ok {
			log.WithFields(logrus.Fields{
				"slot":          slashing.Header_1.Header.Slot,
				"proposerIndex": slashing.Header_1.Header.ProposerIndex,
			}).Errorf("Did not detect simulated proposer slashing")
			continue
		}
		log.WithFields(logrus.Fields{
			"slot":          slashing.Header_1.Header.Slot,
			"proposerIndex": slashing.Header_1.Header.ProposerIndex,
		}).Info("Correctly detected simulated proposer slashing")
	}
	for slashingRoot, slashing := range s.sentAttesterSlashings {
		if _, ok := s.detectedAttesterSlashings[slashingRoot]; !ok {
			log.WithFields(logrus.Fields{
				"targetEpoch":     slashing.Attestation_1.Data.Target.Epoch,
				"prevTargetEpoch": slashing.Attestation_2.Data.Target.Epoch,
				"sourceEpoch":     slashing.Attestation_1.Data.Source.Epoch,
				"prevSourceEpoch": slashing.Attestation_2.Data.Source.Epoch,
			}).Errorf("Did not detect simulated attester slashing")
			continue
		}
		log.WithFields(logrus.Fields{
			"targetEpoch":     slashing.Attestation_1.Data.Target.Epoch,
			"prevTargetEpoch": slashing.Attestation_2.Data.Target.Epoch,
			"sourceEpoch":     slashing.Attestation_1.Data.Source.Epoch,
			"prevSourceEpoch": slashing.Attestation_2.Data.Source.Epoch,
		}).Info("Correctly detected simulated attester slashing")
	}
}

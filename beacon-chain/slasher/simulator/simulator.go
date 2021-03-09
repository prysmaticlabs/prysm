package simulator

import (
	"context"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/slasher"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/sirupsen/logrus"
)

// Parameters for a slasher simulator.
type Parameters struct {
	AggregationPercent     float64
	ProposerSlashingProbab float64
	AttesterSlashingProbab float64
	NumValidators          uint64
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
	sentSlashings         [][32]byte
	detectedSlashings     map[[32]byte]bool
	genesisTime           time.Time
}

// DefaultParams for launching a slasher simulator.
func DefaultParams() *Parameters {
	return &Parameters{
		AggregationPercent:     1.0,
		ProposerSlashingProbab: 0.0,
		AttesterSlashingProbab: 0.0,
		NumValidators:          32,
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
	sentSlashings := [][32]byte{}
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
		"proposerSlashingProbab": s.params.ProposerSlashingProbab,
		"attesterSlashingProbab": s.params.AttesterSlashingProbab,
	}).Info("Starting slasher simulator")
	go s.simulateBlockProposals(s.ctx)
	go s.simulateAttestations(s.ctx)
	go s.receiveSlashings(s.ctx)
	s.slasher.Start()
}

// Stop the simulator.
func (s *Simulator) Stop() error {
	return s.slasher.Stop()
}

func (s *Simulator) simulateBlockProposals(ctx context.Context) {
	ticker := slotutil.NewSlotTicker(s.genesisTime.Add(time.Millisecond*500), params.BeaconConfig().SecondsPerSlot)
	defer ticker.Done()
	for {
		select {
		case slot := <-ticker.C():
			log.Info("Producing block")
			blockHeaders := generateBlockHeadersForSlot(s.params, slot)
			for _, bb := range blockHeaders {
				s.beaconBlocksFeed.Send(bb)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Simulator) simulateAttestations(ctx context.Context) {
	ticker := slotutil.NewSlotTicker(s.genesisTime.Add(time.Millisecond*500), params.BeaconConfig().SecondsPerSlot)
	defer ticker.Done()
	for {
		select {
		case slot := <-ticker.C():
			atts := generateAttestationsForSlot(s.params, slot)
			log.Infof("Producing %d atts for slot %d\n", len(atts), slot)
			for _, aa := range atts {
				s.indexedAttsFeed.Send(aa)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Simulator) receiveSlashings(ctx context.Context) {
	sentSlashings := make(chan *feed.Event, 1)
	s.sentAttSlashingFeed.Subscribe(sentSlashings)
	s.sentBlockSlashingFeed.Subscribe(sentSlashings)
	detectedSlashings := make(chan *feed.Event, 1)
	s.attSlasingFeed.Subscribe(detectedSlashings)
	s.blockSlashingFeed.Subscribe(detectedSlashings)

	for {
		select {
		case event := <-sentSlashings:
			switch event.Type {
			case slashertypes.AttesterSlashing:
				attSlashing, ok := event.Data.(*ethpb.AttesterSlashing)
				if !ok {
					log.Error("not ok")
					return
				}
				slashingRoot, err := attSlashing.HashTreeRoot()
				if err != nil {
					log.Error(err)
					return
				}
				log.Printf("Sent att slashing %#x", slashingRoot)
				s.sentSlashings = append(s.sentSlashings, slashingRoot)
			case slashertypes.ProposerSlashing:
				proposerSlashing, ok := event.Data.(*ethpb.ProposerSlashing)
				if !ok {
					log.Error("not ok")
					return
				}
				slashingRoot, err := proposerSlashing.HashTreeRoot()
				if err != nil {
					log.Error(err)
					return
				}
				log.Printf("Sent att slashing %#x", slashingRoot)
				s.sentSlashings = append(s.sentSlashings, slashingRoot)
			}
		case detectedEvent := <-detectedSlashings:
			switch detectedEvent.Type {
			case slashertypes.AttesterSlashing:
				attSlashing, ok := detectedEvent.Data.(*ethpb.AttesterSlashing)
				if !ok {
					log.Error("not ok")
					return
				}
				slashingRoot, err := attSlashing.HashTreeRoot()
				if err != nil {
					log.Error(err)
					return
				}
				s.logIfDetected(slashingRoot)
				s.detectedSlashings[slashingRoot] = true
			case slashertypes.ProposerSlashing:
				proposerSlashing, ok := detectedEvent.Data.(*ethpb.ProposerSlashing)
				if !ok {
					log.Error("not ok")
					return
				}
				slashingRoot, err := proposerSlashing.HashTreeRoot()
				if err != nil {
					log.Error(err)
					return
				}
				s.logIfDetected(slashingRoot)
				s.detectedSlashings[slashingRoot] = true
			}
		}
	}
}

func (s *Simulator) logIfDetected(slashingRoot [32]byte) {
	for _, ss := range s.sentSlashings {
		if slashingRoot == ss {
			log.Printf("Slashing detected! %#x", slashingRoot)
			return
		}
	}
	log.Printf("No slashing detected for %#x", slashingRoot)
	return
}

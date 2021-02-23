package simulator

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/slasher"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
)

type Parameters struct {
	ProposerSlashingProbab float64
	AttesterSlashingProbab float64
	NumValidators          uint64
}

type Simulator struct {
	ctx              context.Context
	cancel           context.CancelFunc
	slasher          *slasher.Service
	params           *Parameters
	indexedAttsFeed  *event.Feed
	beaconBlocksFeed *event.Feed
	genesisTime      time.Time
}

func DefaultParams() *Parameters {
	return &Parameters{
		ProposerSlashingProbab: 0.5,
		AttesterSlashingProbab: 0.02,
		NumValidators:          16384,
	}
}

func New(ctx context.Context, beaconDB db.Database) (*Simulator, error) {
	indexedAttsFeed := new(event.Feed)
	beaconBlocksFeed := new(event.Feed)
	genesisTime := time.Now()
	slasherSrv, err := slasher.New(ctx, &slasher.ServiceConfig{
		IndexedAttsFeed:  indexedAttsFeed,
		BeaconBlocksFeed: beaconBlocksFeed,
		Database:         beaconDB,
		GenesisTime:      genesisTime,
	})
	if err != nil {
		return nil, err
	}
	return &Simulator{
		ctx:              ctx,
		slasher:          slasherSrv,
		params:           DefaultParams(),
		indexedAttsFeed:  indexedAttsFeed,
		beaconBlocksFeed: beaconBlocksFeed,
		genesisTime:      genesisTime,
	}, nil
}

func (s *Simulator) Start() {
	go s.simulateBlockProposals(s.ctx)
	go s.simulateAttestations(s.ctx)
	s.slasher.Start()
}

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

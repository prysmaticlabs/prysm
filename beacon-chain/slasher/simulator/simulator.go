package simulator

import (
	"context"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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
		AttesterSlashingProbab: 0.5,
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
	ticker := slotutil.NewSlotTicker(s.genesisTime, params.BeaconConfig().SecondsPerSlot)
	defer ticker.Done()
	for {
		select {
		case <-ticker.C():
			log.Info("Producing block")
			s.beaconBlocksFeed.Send(&ethpb.BeaconBlockHeader{
				Slot:          1,
				ProposerIndex: 1,
			})
		case <-ctx.Done():
			return
		}
	}
}

func (s *Simulator) simulateAttestations(ctx context.Context) {
	ticker := slotutil.NewSlotTicker(s.genesisTime, params.BeaconConfig().SecondsPerSlot)
	defer ticker.Done()
	for {
		select {
		case <-ticker.C():
			log.Info("Producing att")
			s.indexedAttsFeed.Send(&ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2, 3},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  make([]byte, 32),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  make([]byte, 32),
					},
				},
			})
		case <-ctx.Done():
			return
		}
	}
}

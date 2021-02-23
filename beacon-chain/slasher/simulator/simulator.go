package simulator

import (
	"context"
	"fmt"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
		case slot := <-ticker.C():
			atts := generateAttestationsForSlot(s.params, slot)
			log.Infof("Producing %d atts for slot %d", len(atts), slot)
			for _, aa := range atts {
				s.indexedAttsFeed.Send(aa)
			}
		case <-ctx.Done():
			return
		}
	}
}

func generateAttestationsForSlot(simParams *Parameters, slot types.Slot) []*ethpb.IndexedAttestation {
	var attestations []*ethpb.IndexedAttestation
	currentEpoch := helpers.SlotToEpoch(slot)

	committeesPerSlot := helpers.SlotCommitteeCount(simParams.NumValidators)
	fmt.Printf("Committees per slot: %d\n", committeesPerSlot)
	attsPerCommittee := params.BeaconConfig().MaxAttestations / committeesPerSlot
	fmt.Printf("Attestations per Committee: %d\n", attsPerCommittee)
	valsPerCommittee := simParams.NumValidators / (committeesPerSlot * uint64(params.BeaconConfig().SlotsPerEpoch))
	fmt.Printf("Validators per Committee: %d\n", valsPerCommittee)

	var sourceEpoch types.Epoch = 0
	if currentEpoch != 0 {
		sourceEpoch = currentEpoch - 1
	}
	for c := types.CommitteeIndex(0); uint64(c) < committeesPerSlot; c++ {
		attData := &ethpb.AttestationData{
			Slot:           slot,
			CommitteeIndex: c,
			Source: &ethpb.Checkpoint{
				Epoch: sourceEpoch,
			},
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
			},
		}

		for i := uint64(0); i < attsPerCommittee; i++ {
			indices := make([]uint64, 0, valsPerCommittee)
			startIdx := i * valsPerCommittee
			if startIdx >= simParams.NumValidators {
				startIdx = 0
			}
			endIdx := (i + 1) * valsPerCommittee
			if endIdx > simParams.NumValidators {
				endIdx = simParams.NumValidators
			}
			for v := startIdx; v < endIdx; v++ {
				indices = append(indices, v)
			}
			att := &ethpb.IndexedAttestation{
				AttestingIndices: indices,
				Data:             attData,
			}
			attestations = append(attestations, att)
		}
	}
	return attestations
}

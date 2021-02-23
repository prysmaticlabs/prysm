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
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
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

func generateBlockHeadersForSlot(simParams *Parameters, slot types.Slot) []*ethpb.BeaconBlockHeader {
	blocks := make([]*ethpb.BeaconBlockHeader, 1)
	proposer := rand.NewGenerator().Uint64() % simParams.NumValidators
	blocks[0] = &ethpb.BeaconBlockHeader{
		Slot:          slot,
		ProposerIndex: proposer,
		BodyRoot:      bytesutil.PadTo([]byte("good block"), 32),
	}
	if rand.NewGenerator().Float64() < simParams.ProposerSlashingProbab {
		fmt.Println("Making a slashable block!")
		blocks = append(blocks, &ethpb.BeaconBlockHeader{
			Slot:          slot,
			ProposerIndex: proposer,
			BodyRoot:      bytesutil.PadTo([]byte("bad block"), 32),
		})
	}
	return blocks
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
			startIdx := i * valsPerCommittee
			if startIdx >= simParams.NumValidators {
				startIdx = 0
			}
			endIdx := (i + 1) * valsPerCommittee
			if endIdx > simParams.NumValidators {
				endIdx = simParams.NumValidators
			}
			indices := make([]uint64, 0, valsPerCommittee)
			for v := startIdx; v < endIdx; v++ {
				indices = append(indices, v)
			}
			att := &ethpb.IndexedAttestation{
				AttestingIndices: indices,
				Data:             attData,
			}
			attestations = append(attestations, att)
			if rand.NewGenerator().Float64() < simParams.AttesterSlashingProbab {
				attestations = append(attestations, makeSlashableFromAtt(att, []uint64{indices[0]}))
			}
		}
	}
	return attestations
}

func makeSlashableFromAtt(att *ethpb.IndexedAttestation, indices []uint64) *ethpb.IndexedAttestation {
	fmt.Println("Making a slashable att!")
	if att.Data.Source.Epoch <= 2 {
		return makeDoubleVoteFromAtt(att, indices)
	}
	attData := &ethpb.AttestationData{
		Slot:           att.Data.Slot,
		CommitteeIndex: att.Data.CommitteeIndex,
		Source: &ethpb.Checkpoint{
			Epoch: att.Data.Source.Epoch - 3,
		},
		Target: &ethpb.Checkpoint{
			Epoch: att.Data.Target.Epoch,
		},
	}
	return &ethpb.IndexedAttestation{
		AttestingIndices: indices,
		Data:             attData,
	}
}

func makeDoubleVoteFromAtt(att *ethpb.IndexedAttestation, indices []uint64) *ethpb.IndexedAttestation {
	attData := &ethpb.AttestationData{
		Slot:            att.Data.Slot,
		CommitteeIndex:  att.Data.CommitteeIndex,
		BeaconBlockRoot: bytesutil.PadTo([]byte("slash me"), 32),
		Source: &ethpb.Checkpoint{
			Epoch: att.Data.Source.Epoch,
		},
		Target: &ethpb.Checkpoint{
			Epoch: att.Data.Target.Epoch,
		},
	}
	return &ethpb.IndexedAttestation{
		AttestingIndices: indices,
		Data:             attData,
	}
}

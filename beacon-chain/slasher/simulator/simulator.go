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
		ParentRoot:    bytesutil.PadTo([]byte{}, 32),
		StateRoot:     bytesutil.PadTo([]byte{}, 32),
		BodyRoot:      bytesutil.PadTo([]byte("good block"), 32),
	}
	if rand.NewGenerator().Float64() < simParams.ProposerSlashingProbab {
		fmt.Println("Making a slashable block!")
		blocks = append(blocks, &ethpb.BeaconBlockHeader{
			Slot:          slot,
			ProposerIndex: proposer,
			ParentRoot:    bytesutil.PadTo([]byte{}, 32),
			StateRoot:     bytesutil.PadTo([]byte{}, 32),
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
	valsPerSlot := committeesPerSlot * valsPerCommittee

	var sourceEpoch types.Epoch = 0
	if currentEpoch != 0 {
		sourceEpoch = currentEpoch - 1
	}

	startIdx := valsPerSlot * uint64(slot%params.BeaconConfig().SlotsPerEpoch)
	endIdx := startIdx + valsPerCommittee
	for c := types.CommitteeIndex(0); uint64(c) < committeesPerSlot; c++ {
		attData := &ethpb.AttestationData{
			Slot:            slot,
			CommitteeIndex:  c,
			BeaconBlockRoot: bytesutil.PadTo([]byte("block"), 32),
			Source: &ethpb.Checkpoint{
				Epoch: sourceEpoch,
				Root:  bytesutil.PadTo([]byte("source"), 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  bytesutil.PadTo([]byte("target"), 32),
			},
		}

		for i := startIdx; i < endIdx; i += valsPerCommittee {
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
				fmt.Printf("Slashing made for index %d\n", indices[0])
				attestations = append(attestations, makeSlashableFromAtt(att, []uint64{indices[0]}))
			}
		}
		startIdx += valsPerCommittee
		endIdx += valsPerCommittee
	}
	return attestations
}

func makeSlashableFromAtt(att *ethpb.IndexedAttestation, indices []uint64) *ethpb.IndexedAttestation {
	if att.Data.Source.Epoch <= 2 {
		return makeDoubleVoteFromAtt(att, indices)
	}
	attData := &ethpb.AttestationData{
		Slot:            att.Data.Slot,
		CommitteeIndex:  att.Data.CommitteeIndex,
		BeaconBlockRoot: att.Data.BeaconBlockRoot,
		Source: &ethpb.Checkpoint{
			Epoch: att.Data.Source.Epoch - 3,
			Root:  att.Data.Source.Root,
		},
		Target: &ethpb.Checkpoint{
			Epoch: att.Data.Target.Epoch,
			Root:  att.Data.Target.Root,
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
			Root:  att.Data.Source.Root,
		},
		Target: &ethpb.Checkpoint{
			Epoch: att.Data.Target.Epoch,
			Root:  att.Data.Target.Root,
		},
	}
	return &ethpb.IndexedAttestation{
		AttestingIndices: indices,
		Data:             attData,
	}
}

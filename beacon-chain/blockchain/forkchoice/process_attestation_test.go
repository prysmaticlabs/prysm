package forkchoice

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestStore_OnAttestation(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	_, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	BlkWithOutState := &ethpb.BeaconBlock{Slot: 0}
	if err := db.SaveBlock(ctx, BlkWithOutState); err != nil {
		t.Fatal(err)
	}
	BlkWithOutStateRoot, _ := ssz.SigningRoot(BlkWithOutState)

	BlkWithStateBadAtt := &ethpb.BeaconBlock{Slot: 1}
	if err := db.SaveBlock(ctx, BlkWithStateBadAtt); err != nil {
		t.Fatal(err)
	}
	BlkWithStateBadAttRoot, _ := ssz.SigningRoot(BlkWithStateBadAtt)
	if err := store.db.SaveState(ctx, &pb.BeaconState{}, BlkWithStateBadAttRoot); err != nil {
		t.Fatal(err)
	}

	BlkWithValidState := &ethpb.BeaconBlock{Slot: 2}
	if err := db.SaveBlock(ctx, BlkWithValidState); err != nil {
		t.Fatal(err)
	}
	BlkWithValidStateRoot, _ := ssz.SigningRoot(BlkWithValidState)
	if err := store.db.SaveState(ctx, &pb.BeaconState{
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}, BlkWithValidStateRoot); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		a             *ethpb.Attestation
		s             *pb.BeaconState
		wantErr       bool
		wantErrString string
	}{
		{
			name:          "attestation's target root not in db",
			a:             &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Root: []byte{'A'}}}},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "target root 0x41 does not exist in db",
		},
		{
			name:          "no pre state for attestations's target block",
			a:             &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Root: BlkWithOutStateRoot[:]}}},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "pre state of target block 0 does not exist",
		},
		{
			name: "process attestation from future epoch",
			a: &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Epoch: params.BeaconConfig().FarFutureEpoch,
				Root: BlkWithStateBadAttRoot[:]}}},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "could not process attestation from the future epoch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
				t.Fatal(err)
			}

			_, err := store.OnAttestation(ctx, tt.a)
			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.wantErrString) {
					t.Errorf("Store.OnAttestation() error = %v, wantErr = %v", err, tt.wantErrString)
				}
			} else {
				t.Error(err)
			}
		})
	}
}

func TestStore_SaveCheckpointState(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	params.UseDemoBeaconConfig()

	store := NewForkChoiceService(ctx, db)

	crosslinks := make([]*ethpb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &ethpb.Crosslink{
			ParentRoot: make([]byte, 32),
			DataRoot:   make([]byte, 32),
		}
	}
	s := &pb.BeaconState{
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:           make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		StateRoots:                 make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		BlockRoots:                 make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
		LatestBlockHeader:          &ethpb.BeaconBlockHeader{},
		JustificationBits:          []byte{0},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{},
		CurrentCrosslinks:          crosslinks,
		CompactCommitteesRoots:     make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Slashings:                  make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		FinalizedCheckpoint:        &ethpb.Checkpoint{},
	}
	if err := store.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
		t.Fatal(err)
	}

	cp1 := &ethpb.Checkpoint{Epoch: 1, Root: []byte{'A'}}
	s1, err := store.saveCheckpointState(ctx, s, cp1)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Slot != 1*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot)
	}

	cp2 := &ethpb.Checkpoint{Epoch: 2, Root: []byte{'B'}}
	s2, err := store.saveCheckpointState(ctx, s, cp2)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Slot != 2*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 2*params.BeaconConfig().SlotsPerEpoch, s2.Slot)
	}

	s1, err = store.saveCheckpointState(ctx, nil, cp1)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Slot != 1*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot)
	}

	s1, err = store.checkpointState.StateByCheckpoint(cp1)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Slot != 1*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot)
	}

	s2, err = store.checkpointState.StateByCheckpoint(cp2)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Slot != 2*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 2*params.BeaconConfig().SlotsPerEpoch, s2.Slot)
	}

	s.Slot = params.BeaconConfig().SlotsPerEpoch + 1
	if err := store.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
		t.Fatal(err)
	}
	cp3 := &ethpb.Checkpoint{Epoch: 1, Root: []byte{'C'}}
	s3, err := store.saveCheckpointState(ctx, s, cp3)
	if err != nil {
		t.Fatal(err)
	}
	if s3.Slot != s.Slot {
		t.Errorf("Wanted state slot: %d, got: %d", s.Slot, s3.Slot)
	}
}

func TestStore_AggregateAttestation(t *testing.T) {
	store := &Store{attsQueue: make(map[[32]byte]*ethpb.Attestation)}

	bits := bitfield.NewBitlist(8)
	bits.SetBitAt(0, true)
	a := &ethpb.Attestation{Data: &ethpb.AttestationData{}, AggregationBits: bits}

	if err := store.aggregateAttestation(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	r, _ := ssz.HashTreeRoot(a.Data)
	if !bytes.Equal(store.attsQueue[r].AggregationBits, bits) {
		t.Error("Received incorrect aggregation bitfield")
	}

	bits.SetBitAt(1, true)
	a = &ethpb.Attestation{Data: &ethpb.AttestationData{}, AggregationBits: bits}
	if err := store.aggregateAttestation(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(store.attsQueue[r].AggregationBits, []byte{3, 1}) {
		t.Error("Received incorrect aggregation bitfield")
	}

	bits.SetBitAt(7, true)
	a = &ethpb.Attestation{Data: &ethpb.AttestationData{}, AggregationBits: bits}
	if err := store.aggregateAttestation(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(store.attsQueue[r].AggregationBits, []byte{131, 1}) {
		t.Error("Received incorrect aggregation bitfield")
	}
}

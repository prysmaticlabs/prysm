package forkchoice

import (
	"context"
	"strings"
	"testing"

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
		{
			name: "process attestation with invalid index",
			a: &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Epoch: 0, Root: BlkWithStateBadAttRoot[:]},
				Crosslink: &ethpb.Crosslink{}}},
			s:             &pb.BeaconState{Slot: 1},
			wantErr:       true,
			wantErrString: "could not convert attestation to indexed attestation",
		},
		{
			name: "process attestation with invalid signature",
			a: &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Epoch: 0, Root: BlkWithValidStateRoot[:]},
				Crosslink: &ethpb.Crosslink{}}},
			s:             &pb.BeaconState{Slot: 1},
			wantErr:       true,
			wantErrString: "could not verify indexed attestation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.GenesisStore(ctx, tt.s); err != nil {
				t.Fatal(err)
			}

			err := store.OnAttestation(ctx, tt.a)
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

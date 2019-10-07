package forkchoice

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStore_OnBlock(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	randomParentRoot := []byte{'a'}
	if err := store.db.SaveState(ctx, &pb.BeaconState{}, bytesutil.ToBytes32(randomParentRoot)); err != nil {
		t.Fatal(err)
	}
	randomParentRoot2 := roots[1]
	if err := store.db.SaveState(ctx, &pb.BeaconState{}, bytesutil.ToBytes32(randomParentRoot2)); err != nil {
		t.Fatal(err)
	}
	validGenesisRoot := []byte{'g'}
	if err := store.db.SaveState(ctx, &pb.BeaconState{}, bytesutil.ToBytes32(validGenesisRoot)); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		blk           *ethpb.BeaconBlock
		s             *pb.BeaconState
		time          uint64
		wantErrString string
	}{
		{
			name:          "parent block root does not have a state",
			blk:           &ethpb.BeaconBlock{},
			s:             &pb.BeaconState{},
			wantErrString: "pre state of slot 0 does not exist",
		},
		{
			name:          "block is from the feature",
			blk:           &ethpb.BeaconBlock{ParentRoot: randomParentRoot, Slot: params.BeaconConfig().FarFutureEpoch},
			s:             &pb.BeaconState{},
			wantErrString: "could not process slot from the future",
		},
		{
			name:          "could not get finalized block",
			blk:           &ethpb.BeaconBlock{ParentRoot: randomParentRoot},
			s:             &pb.BeaconState{},
			wantErrString: "block from slot 0 is not a descendent of the current finalized block",
		},
		{
			name:          "same slot as finalized block",
			blk:           &ethpb.BeaconBlock{Slot: 0, ParentRoot: randomParentRoot2},
			s:             &pb.BeaconState{},
			wantErrString: "block is equal or earlier than finalized block, slot 0 < slot 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
				t.Fatal(err)
			}
			store.finalizedCheckpt.Root = roots[0]

			err := store.OnBlock(ctx, tt.blk)
			if !strings.Contains(err.Error(), tt.wantErrString) {
				t.Errorf("Store.OnBlock() error = %v, wantErr = %v", err, tt.wantErrString)
			}
		})
	}
}

func TestStore_SaveNewValidators(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)
	preCount := 2 // validators 0 and validators 1
	s := &pb.BeaconState{Validators: []*ethpb.Validator{
		{PublicKey: []byte{0}}, {PublicKey: []byte{1}},
		{PublicKey: []byte{2}}, {PublicKey: []byte{3}},
	}}
	if err := store.saveNewValidators(ctx, preCount, s); err != nil {
		t.Fatal(err)
	}

	if !db.HasValidatorIndex(ctx, bytesutil.ToBytes48([]byte{2})) {
		t.Error("Wanted validator saved in db")
	}
	if !db.HasValidatorIndex(ctx, bytesutil.ToBytes48([]byte{3})) {
		t.Error("Wanted validator saved in db")
	}
	if db.HasValidatorIndex(ctx, bytesutil.ToBytes48([]byte{1})) {
		t.Error("validator not suppose to be saved in db")
	}
}

func TestStore_UpdateBlockAttestationVote(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	params.UseMinimalConfig()

	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}

	store := NewForkChoiceService(ctx, db)
	r := [32]byte{'A'}
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: r[:]},
			Crosslink: &ethpb.Crosslink{
				Shard:      0,
				StartEpoch: 0,
			},
		},
		AggregationBits: []byte{255},
		CustodyBits:     []byte{255},
	}
	if err := store.db.SaveState(ctx, beaconState, r); err != nil {
		t.Fatal(err)
	}

	indices, err := blocks.ConvertToIndexed(beaconState, att)
	if err != nil {
		t.Fatal(err)
	}

	var attestedIndices []uint64
	for _, k := range append(indices.CustodyBit_0Indices, indices.CustodyBit_1Indices...) {
		attestedIndices = append(attestedIndices, k)
	}

	if err := store.updateBlockAttestationVote(ctx, att); err != nil {
		t.Fatal(err)
	}
	for _, i := range attestedIndices {
		v, err := store.db.ValidatorLatestVote(ctx, i)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v.Root, r[:]) {
			t.Error("Attested roots don't match")
		}
	}
}

func TestStore_UpdateBlockAttestationsVote(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	params.UseMinimalConfig()

	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}

	store := NewForkChoiceService(ctx, db)
	r := [32]byte{'A'}
	atts := make([]*ethpb.Attestation, 5)
	hashes := make([][32]byte, 5)
	for i := 0; i < len(atts); i++ {
		atts[i] = &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]},
				Target: &ethpb.Checkpoint{Epoch: 0, Root: r[:]},
				Crosslink: &ethpb.Crosslink{
					Shard:      uint64(i),
					StartEpoch: 0,
				},
			},
			AggregationBits: []byte{255},
			CustodyBits:     []byte{255},
		}
		h, _ := hashutil.HashProto(atts[i])
		hashes[i] = h
	}

	if err := store.db.SaveState(ctx, beaconState, r); err != nil {
		t.Fatal(err)
	}

	if err := store.updateBlockAttestationsVotes(ctx, atts); err != nil {
		t.Fatal(err)
	}

	for _, h := range hashes {
		if !store.seenAtts[h] {
			t.Error("Seen attestation did not get recorded")
		}
	}
}

func TestStore_SavesNewBlockAttestations(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)
	a1 := &ethpb.Attestation{Data: &ethpb.AttestationData{}, AggregationBits: bitfield.Bitlist{0x02}}
	a2 := &ethpb.Attestation{Data: &ethpb.AttestationData{BeaconBlockRoot: []byte{'A'}}, AggregationBits: bitfield.Bitlist{0x02}}
	r1, _ := ssz.HashTreeRoot(a1.Data)
	r2, _ := ssz.HashTreeRoot(a2.Data)

	store.saveNewBlockAttestations(ctx, []*ethpb.Attestation{a1, a2})

	saved, err := store.db.Attestation(ctx, r1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a1, saved) {
		t.Error("did not retrieve saved attestation")
	}

	saved, err = store.db.Attestation(ctx, r2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a2, saved) {
		t.Error("did not retrieve saved attestation")
	}

	a1 = &ethpb.Attestation{Data: &ethpb.AttestationData{}, AggregationBits: bitfield.Bitlist{0x03}}
	a2 = &ethpb.Attestation{Data: &ethpb.AttestationData{BeaconBlockRoot: []byte{'A'}}, AggregationBits: bitfield.Bitlist{0x03}}

	store.saveNewBlockAttestations(ctx, []*ethpb.Attestation{a1, a2})

	saved, err = store.db.Attestation(ctx, r1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a1, saved) {
		t.Error("did not retrieve saved attestation")
	}

	saved, err = store.db.Attestation(ctx, r2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a2, saved) {
		t.Error("did not retrieve saved attestation")
	}
}

package forkchoice

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	store := NewForkChoiceService(ctx, db)
	r := [32]byte{'A'}
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: r[:]},
		},
		AggregationBits: []byte{255},
		CustodyBits:     []byte{255},
	}
	if err := store.db.SaveState(ctx, beaconState, r); err != nil {
		t.Fatal(err)
	}

	indices, err := blocks.ConvertToIndexed(ctx, beaconState, att)
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
		v := store.latestVoteMap[i]
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
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
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
	a1 := &ethpb.Attestation{Data: &ethpb.AttestationData{}, AggregationBits: bitfield.Bitlist{0b101}, CustodyBits: bitfield.NewBitlist(2)}
	a2 := &ethpb.Attestation{Data: &ethpb.AttestationData{BeaconBlockRoot: []byte{'A'}}, AggregationBits: bitfield.Bitlist{0b110}, CustodyBits: bitfield.NewBitlist(2)}
	r1, _ := ssz.HashTreeRoot(a1.Data)
	r2, _ := ssz.HashTreeRoot(a2.Data)

	if err := store.saveNewBlockAttestations(ctx, []*ethpb.Attestation{a1, a2}); err != nil {
		t.Fatal(err)
	}

	saved, err := store.db.AttestationsByDataRoot(ctx, r1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]*ethpb.Attestation{a1}, saved) {
		t.Error("did not retrieve saved attestation")
	}

	saved, err = store.db.AttestationsByDataRoot(ctx, r2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]*ethpb.Attestation{a2}, saved) {
		t.Error("did not retrieve saved attestation")
	}

	a1 = &ethpb.Attestation{Data: &ethpb.AttestationData{}, AggregationBits: bitfield.Bitlist{0b111}, CustodyBits: bitfield.NewBitlist(2)}
	a2 = &ethpb.Attestation{Data: &ethpb.AttestationData{BeaconBlockRoot: []byte{'A'}}, AggregationBits: bitfield.Bitlist{0b111}, CustodyBits: bitfield.NewBitlist(2)}

	if err := store.saveNewBlockAttestations(ctx, []*ethpb.Attestation{a1, a2}); err != nil {
		t.Fatal(err)
	}

	saved, err = store.db.AttestationsByDataRoot(ctx, r1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]*ethpb.Attestation{a1}, saved) {
		t.Error("did not retrieve saved attestation")
	}

	saved, err = store.db.AttestationsByDataRoot(ctx, r2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]*ethpb.Attestation{a2}, saved) {
		t.Error("did not retrieve saved attestation")
	}
}

func TestRemoveStateSinceLastFinalized(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()

	store := NewForkChoiceService(ctx, db)

	// Save 100 blocks in DB, each has a state.
	numBlocks := 100
	totalBlocks := make([]*ethpb.BeaconBlock, numBlocks)
	blockRoots := make([][32]byte, 0)
	for i := 0; i < len(totalBlocks); i++ {
		totalBlocks[i] = &ethpb.BeaconBlock{
			Slot: uint64(i),
		}
		r, err := ssz.SigningRoot(totalBlocks[i])
		if err != nil {
			t.Fatal(err)
		}
		if err := store.db.SaveState(ctx, &pb.BeaconState{Slot: uint64(i)}, r); err != nil {
			t.Fatal(err)
		}
		if err := store.db.SaveBlock(ctx, totalBlocks[i]); err != nil {
			t.Fatal(err)
		}
		blockRoots = append(blockRoots, r)
	}

	// New finalized epoch: 1
	finalizedEpoch := uint64(1)
	finalizedSlot := finalizedEpoch * params.BeaconConfig().SlotsPerEpoch
	endSlot := helpers.StartSlot(finalizedEpoch+1) - 1 // Inclusive
	if err := store.rmStatesOlderThanLastFinalized(ctx, 0, endSlot); err != nil {
		t.Fatal(err)
	}
	for _, r := range blockRoots {
		s, err := store.db.State(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		// Also verifies genesis state didnt get deleted
		if s != nil && s.Slot != finalizedSlot && s.Slot != 0 && s.Slot < endSlot {
			t.Errorf("State with slot %d should not be in DB", s.Slot)
		}
	}

	// New finalized epoch: 5
	newFinalizedEpoch := uint64(5)
	newFinalizedSlot := newFinalizedEpoch * params.BeaconConfig().SlotsPerEpoch
	endSlot = helpers.StartSlot(newFinalizedEpoch+1) - 1 // Inclusive
	if err := store.rmStatesOlderThanLastFinalized(ctx, helpers.StartSlot(finalizedEpoch+1)-1, endSlot); err != nil {
		t.Fatal(err)
	}
	for _, r := range blockRoots {
		s, err := store.db.State(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		// Also verifies genesis state didnt get deleted
		if s != nil && s.Slot != newFinalizedSlot && s.Slot != finalizedSlot && s.Slot != 0 && s.Slot < endSlot {
			t.Errorf("State with slot %d should not be in DB", s.Slot)
		}
	}
}

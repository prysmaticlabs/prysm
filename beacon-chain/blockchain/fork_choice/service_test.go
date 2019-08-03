package forkchoice

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// blockTree constructs the following tree
//    /- B1
// B0           /- B5 - B7
//    \- B3 - B4 - B6 - B8
// (B1, and B3 are all from the same slots)
func blockTree(db *db.BeaconDB) ([][]byte, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: []byte{'g'}}
	r0, _ := ssz.SigningRoot(b0)
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, _ := ssz.SigningRoot(b1)
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: r0[:]}
	r3, _ := ssz.SigningRoot(b3)
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: r3[:]}
	r4, _ := ssz.SigningRoot(b4)
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: r4[:]}
	r5, _ := ssz.SigningRoot(b5)
	b6 := &ethpb.BeaconBlock{Slot: 6, ParentRoot: r4[:]}
	r6, _ := ssz.SigningRoot(b6)
	b7 := &ethpb.BeaconBlock{Slot: 7, ParentRoot: r5[:]}
	r7, _ := ssz.SigningRoot(b7)
	b8 := &ethpb.BeaconBlock{Slot: 8, ParentRoot: r6[:]}
	r8, _ := ssz.SigningRoot(b8)
	for _, b := range []*ethpb.BeaconBlock{b0, b1, b3, b4, b5, b6, b7, b8} {
		if err := db.SaveBlock(b); err != nil {
			return nil, err
		}
		if err := db.SaveForkChoiceState(context.Background(), &pb.BeaconState{}, b.ParentRoot); err != nil {
			return nil, err
		}
	}
	return [][]byte{r0[:], r1[:], nil, r3[:], r4[:], r5[:], r6[:], r7[:], r8[:]}, nil
}

func TestStore_GensisStoreOk(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	genesisTime := uint64(9999)
	genesisState := &pb.BeaconState{GenesisTime: genesisTime}
	genesisStateRoot, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		t.Fatal(err)
	}

	genesisBlk := blocks.NewGenesisBlock(genesisStateRoot[:])
	genesisBlkRoot, err := ssz.SigningRoot(genesisBlk)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.GensisStore(genesisState); err != nil {
		t.Fatal(err)
	}

	if store.time != genesisTime {
		t.Errorf("Wanted time %d, got time %d", genesisTime, store.time)
	}

	genesisCheckpt := &ethpb.Checkpoint{Epoch: 0, Root: genesisBlkRoot[:]}
	if !reflect.DeepEqual(store.justifiedCheckpt, genesisCheckpt) {
		t.Error("Justified check point from genesis store did not match")
	}
	if !reflect.DeepEqual(store.finalizedCheckpt, genesisCheckpt) {
		t.Error("Finalized check point from genesis store did not match")
	}

	b, err := store.db.Block(genesisBlkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b, genesisBlk) {
		t.Error("Incorrect genesis block saved from store")
	}

	s, err := store.db.CheckpointState(ctx, genesisCheckpt)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, genesisState) {
		t.Error("Incorrect genesis state saved from store")
	}
}

func TestStore_AncestorOk(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree(db)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		root []byte
		slot uint64
	}

	//    /- B1
	// B0           /- B5 - B7
	//    \- B3 - B4 - B6 - B8
	tests := []struct {
		args *args
		want []byte
	}{
		{args: &args{roots[1], 0}, want: roots[0]},
		{args: &args{roots[8], 0}, want: roots[0]},
		{args: &args{roots[8], 4}, want: roots[4]},
		{args: &args{roots[7], 4}, want: roots[4]},
		{args: &args{roots[7], 0}, want: roots[0]},
	}
	for _, tt := range tests {
		got, err := store.Ancestor(tt.args.root, tt.args.slot)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Store.Ancestor() = %v, want %v", got, tt.want)
		}
	}
}

func TestStore_AncestorNotPartOfTheChain(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree(db)
	if err != nil {
		t.Fatal(err)
	}

	//    /- B1
	// B0           /- B5 - B7
	//    \- B3 - B4 - B6 - B8
	root, err := store.Ancestor(roots[8], 1)
	if err != nil {
		t.Fatal(err)
	}
	if root != nil {
		t.Error("block at slot 1 is not part of the chain")
	}
	root, err = store.Ancestor(roots[8], 2)
	if err != nil {
		t.Fatal(err)
	}
	if root != nil {
		t.Error("block at slot 2 is not part of the chain")
	}
}

func TestStore_LatestAttestingBalance(t *testing.T) {
	helpers.ClearAllCaches()
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree(db)
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*ethpb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
	}

	s := &pb.BeaconState{Validators: validators}

	if err := store.GensisStore(s); err != nil {
		t.Fatal(err)
	}

	//    /- B1 (33 votes)
	// B0           /- B5 - B7 (33 votes)
	//    \- B3 - B4 - B6 - B8 (34 votes)
	for i := 0; i < len(validators); i++ {
		switch {
		case i < 33:
			if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[1]}); err != nil {
				t.Fatal(err)
			}
		case i > 66:
			if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[7]}); err != nil {
				t.Fatal(err)
			}
		default:
			if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[8]}); err != nil {
				t.Fatal(err)
			}
		}
	}

	tests := []struct {
		root []byte
		want uint64
	}{
		{root: roots[0], want: 100 * 1e9},
		{root: roots[1], want: 33 * 1e9},
		{root: roots[3], want: 67 * 1e9},
		{root: roots[4], want: 67 * 1e9},
		{root: roots[7], want: 33 * 1e9},
		{root: roots[8], want: 34 * 1e9},
	}
	for _, tt := range tests {
		got, err := store.LatestAttestingBalance(tt.root)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("Store.LatestAttestingBalance() = %v, want %v", got, tt.want)
		}
	}
}

func TestStore_ChildrenBlocksFromParentRoot(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree(db)
	if err != nil {
		t.Fatal(err)
	}

	children, err := store.db.ChildrenBlocksFromParent(roots[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(children, [][]byte{roots[1], roots[3]}) {
		t.Error("Did not receive correct children roots")
	}

	children, err = store.db.ChildrenBlocksFromParent(roots[0], 2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(children, [][]byte{roots[3]}) {
		t.Error("Did not receive correct children roots")
	}
}

func TestStore_GetHead(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree(db)
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*ethpb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
	}

	s := &pb.BeaconState{Validators: validators}
	if err := store.GensisStore(s); err != nil {
		t.Fatal(err)
	}
	store.justifiedCheckpt.Root = roots[0]
	if err := store.db.SaveCheckpointState(ctx, s, store.justifiedCheckpt); err != nil {
		t.Fatal(err)
	}

	//    /- B1 (33 votes)
	// B0           /- B5 - B7 (33 votes)
	//    \- B3 - B4 - B6 - B8 (34 votes)
	for i := 0; i < len(validators); i++ {
		switch {
		case i < 33:
			if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[1]}); err != nil {
				t.Fatal(err)
			}
		case i > 66:
			if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[7]}); err != nil {
				t.Fatal(err)
			}
		default:
			if err := store.db.SaveLatestMessage(ctx, uint64(i), &pb.LatestMessage{Root: roots[8]}); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Default head is B8
	head, err := store.Head()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, roots[8]) {
		t.Error("Incorrect head")
	}

	// 1 validator switches vote to B7 to gain 34%, enough to switch head
	if err := store.db.SaveLatestMessage(ctx, 50, &pb.LatestMessage{Root: roots[7]}); err != nil {
		t.Fatal(err)
	}
	head, err = store.Head()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, roots[7]) {
		t.Error("Incorrect head")
	}

	// 18 validators switches vote to B1 to gain 51%, enough to switch head
	for i := 0; i < 18; i++ {
		idx := 50 + uint64(i)
		if err := store.db.SaveLatestMessage(ctx, idx, &pb.LatestMessage{Root: roots[1]}); err != nil {
			t.Fatal(err)
		}
	}
	head, err = store.Head()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, roots[1]) {
		t.Log(head)
		t.Error("Incorrect head")
	}
}

func TestStore_OnBlockErrors(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree(db)
	if err != nil {
		t.Fatal(err)
	}

	randaomParentRoot := []byte{'a'}
	if err := store.db.SaveForkChoiceState(ctx, &pb.BeaconState{}, randaomParentRoot); err != nil {
		t.Fatal(err)
	}
	validGenesisRoot := []byte{'g'}
	if err := store.db.SaveForkChoiceState(ctx, &pb.BeaconState{}, validGenesisRoot); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		blk           *ethpb.BeaconBlock
		s             *pb.BeaconState
		time          uint64
		wantErr       bool
		wantErrString string
	}{
		{
			name:          "parent block root does not have a state",
			blk:           &ethpb.BeaconBlock{},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "pre state of slot 0 does not exist",
		},
		{
			name:          "block is from the feature",
			blk:           &ethpb.BeaconBlock{ParentRoot: randaomParentRoot, Slot: 1},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "could not process block from the future, slot time 6 > current time 0",
		},
		{
			name:          "could not get finalized block",
			blk:           &ethpb.BeaconBlock{ParentRoot: randaomParentRoot},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "block from slot 0 is not a descendent of the current finalized block",
		},
		{
			name:          "same slot as finalized block",
			blk:           &ethpb.BeaconBlock{Slot: 0, ParentRoot: validGenesisRoot},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "block is equal or earlier than finalized block, slot 0 < slot 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.GensisStore(tt.s); err != nil {
				t.Fatal(err)
			}
			store.finalizedCheckpt.Root = roots[0]
			store.time = tt.time

			err := store.OnBlock(tt.blk)
			if tt.wantErr {
				if err.Error() != tt.wantErrString {
					t.Errorf("Store.OnBlock() error = %v, wantErr = %v", err, tt.wantErrString)
				}
			} else {
				t.Error(err)
			}
		})
	}
}

func TestStore_OnAttestationErrors(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	_, err := blockTree(db)
	if err != nil {
		t.Fatal(err)
	}

	BlkWithOutState := &ethpb.BeaconBlock{}
	if err := db.SaveBlock(BlkWithOutState); err != nil {
		t.Fatal(err)
	}
	BlkWithOutStateRoot, _ := ssz.SigningRoot(BlkWithOutState)

	BlkWithStateBadAtt := &ethpb.BeaconBlock{Slot: 1}
	if err := db.SaveBlock(BlkWithStateBadAtt); err != nil {
		t.Fatal(err)
	}
	BlkWithStateBadAttRoot, _ := ssz.SigningRoot(BlkWithStateBadAtt)
	if err := store.db.SaveForkChoiceState(ctx, &pb.BeaconState{}, BlkWithStateBadAttRoot[:]); err != nil {
		t.Fatal(err)
	}

	BlkWithValidState := &ethpb.BeaconBlock{Slot: 2}
	if err := db.SaveBlock(BlkWithValidState); err != nil {
		t.Fatal(err)
	}
	BlkWithValidStateRoot, _ := ssz.SigningRoot(BlkWithValidState)
	if err := store.db.SaveForkChoiceState(ctx, &pb.BeaconState{
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}, BlkWithValidStateRoot[:]); err != nil {
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
			name:    "process attestation from future epoch",
			a:       &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Epoch: 1, Root: BlkWithStateBadAttRoot[:]}}},
			s:       &pb.BeaconState{},
			wantErr: true,
			wantErrString: fmt.Sprintf("could not process attestation from the future epoch, time %d > time 0",
				params.BeaconConfig().SlotsPerEpoch*params.BeaconConfig().SecondsPerSlot),
		},
		{
			name: "process attestation before inclusion delay",
			a: &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Epoch: 0, Root: BlkWithStateBadAttRoot[:]},
				Crosslink: &ethpb.Crosslink{}}},
			s:       &pb.BeaconState{},
			wantErr: true,
			wantErrString: fmt.Sprintf("could not process attestation for fork choice until inclusion delay, time %d > time 0",
				params.BeaconConfig().SecondsPerSlot),
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
			if err := store.GensisStore(tt.s); err != nil {
				t.Fatal(err)
			}

			store.OnTick(tt.s.Slot * params.BeaconConfig().SecondsPerSlot)

			err := store.OnAttestation(tt.a)
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

func TestDeleteValidatorIdx_DeleteWorks(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	epoch := uint64(2)
	v.InsertActivatedIndices(epoch+1, []uint64{0, 1, 2})
	v.InsertExitedVal(epoch+1, []uint64{0, 2})
	var validators []*ethpb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		Validators: validators,
		Slot:       epoch * params.BeaconConfig().SlotsPerEpoch,
	}

	if err := saveValidatorIdx(state, db); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}
	if err := deleteValidatorIdx(state, db); err != nil {
		t.Fatalf("Could not delete validator idx: %v", err)
	}
	wantedIdx := uint64(1)
	idx, err := db.ValidatorIndex(validators[wantedIdx].PublicKey)
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	wantedIdx = uint64(2)
	if db.HasValidator(validators[wantedIdx].PublicKey) {
		t.Errorf("Validator index %d should have been deleted", wantedIdx)
	}
	if v.ExitedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}

func TestSaveValidatorIdx_SaveRetrieveWorks(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	epoch := uint64(1)
	v.InsertActivatedIndices(epoch+1, []uint64{0, 1, 2})
	var validators []*ethpb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		Validators: validators,
		Slot:       epoch * params.BeaconConfig().SlotsPerEpoch,
	}

	if err := saveValidatorIdx(state, db); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}

	wantedIdx := uint64(2)
	idx, err := db.ValidatorIndex(validators[wantedIdx].PublicKey)
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	if v.ActivatedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}

func TestSaveValidatorIdx_IdxNotInState(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	epoch := uint64(100)

	// Tried to insert 5 active indices to DB with only 3 validators in state
	v.InsertActivatedIndices(epoch+1, []uint64{0, 1, 2, 3, 4})
	var validators []*ethpb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		Validators: validators,
		Slot:       epoch * params.BeaconConfig().SlotsPerEpoch,
	}

	if err := saveValidatorIdx(state, db); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}

	wantedIdx := uint64(2)
	idx, err := db.ValidatorIndex(validators[wantedIdx].PublicKey)
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	if v.ActivatedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}

	// Verify the skipped validators are included in the next epoch
	if !reflect.DeepEqual(v.ActivatedValFromEpoch(epoch+2), []uint64{3, 4}) {
		t.Error("Did not get wanted validator from activation queue")
	}
}

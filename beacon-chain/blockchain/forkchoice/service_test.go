package forkchoice

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestStore_GensisStoreOk(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

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

	b, err := store.db.Block(ctx, genesisBlkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b, genesisBlk) {
		t.Error("Incorrect genesis block saved from store")
	}

	h, err := hashutil.HashProto(genesisCheckpt)
	if err != nil {
		t.Fatal(err)
	}
	if store.checkptBlkRoot[h] != genesisBlkRoot {
		t.Error("Incorrect genesis check point to block root saved from store")
	}
}

func TestStore_AncestorOk(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
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
		got, err := store.ancestor(tt.args.root, tt.args.slot)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Store.ancestor() = %v, want %v", got, tt.want)
		}
	}
}

func TestStore_AncestorNotPartOfTheChain(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	//    /- B1
	// B0           /- B5 - B7
	//    \- B3 - B4 - B6 - B8
	root, err := store.ancestor(roots[8], 1)
	if err != nil {
		t.Fatal(err)
	}
	if root != nil {
		t.Error("block at slot 1 is not part of the chain")
	}
	root, err = store.ancestor(roots[8], 2)
	if err != nil {
		t.Fatal(err)
	}
	if root != nil {
		t.Error("block at slot 2 is not part of the chain")
	}
}

func TestStore_LatestAttestingBalance(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
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
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[1]}); err != nil {
				t.Fatal(err)
			}
		case i > 66:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[7]}); err != nil {
				t.Fatal(err)
			}
		default:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[8]}); err != nil {
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
		got, err := store.latestAttestingBalance(tt.root)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("Store.latestAttestingBalance() = %v, want %v", got, tt.want)
		}
	}
}

func TestStore_ChildrenBlocksFromParentRoot(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	filter := filters.NewFilter().SetParentRoot(roots[0]).SetStartSlot(0)
	children, err := store.db.BlockRoots(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(children, [][]byte{roots[1], roots[3]}) {
		t.Error("Did not receive correct children roots")
	}

	filter = filters.NewFilter().SetParentRoot(roots[0]).SetStartSlot(2)
	children, err = store.db.BlockRoots(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(children, [][]byte{roots[3]}) {
		t.Error("Did not receive correct children roots")
	}
}

func TestStore_GetHead(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
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
	if err := store.db.SaveState(ctx, s, bytesutil.ToBytes32(roots[0])); err != nil {
		t.Fatal(err)
	}
	store.justifiedCheckpt.Root = roots[0]
	h, err := hashutil.HashProto(store.justifiedCheckpt)
	if err != nil {
		t.Fatal(err)
	}
	store.checkptBlkRoot[h] = bytesutil.ToBytes32(roots[0])

	//    /- B1 (33 votes)
	// B0           /- B5 - B7 (33 votes)
	//    \- B3 - B4 - B6 - B8 (34 votes)
	for i := 0; i < len(validators); i++ {
		switch {
		case i < 33:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[1]}); err != nil {
				t.Fatal(err)
			}
		case i > 66:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[7]}); err != nil {
				t.Fatal(err)
			}
		default:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[8]}); err != nil {
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
	if err := store.db.SaveValidatorLatestVote(ctx, 50, &pb.ValidatorLatestVote{Root: roots[7]}); err != nil {
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
		if err := store.db.SaveValidatorLatestVote(ctx, idx, &pb.ValidatorLatestVote{Root: roots[1]}); err != nil {
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
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	randaomParentRoot := []byte{'a'}
	if err := store.db.SaveState(ctx, &pb.BeaconState{}, bytesutil.ToBytes32(randaomParentRoot)); err != nil {
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
			blk:           &ethpb.BeaconBlock{ParentRoot: randaomParentRoot, Slot: 1},
			s:             &pb.BeaconState{},
			wantErrString: "could not process block from the future, slot time 6 > current time 0",
		},
		{
			name:          "could not get finalized block",
			blk:           &ethpb.BeaconBlock{ParentRoot: randaomParentRoot},
			s:             &pb.BeaconState{},
			wantErrString: "block from slot 0 is not a descendent of the current finalized block",
		},
		{
			name:          "same slot as finalized block",
			blk:           &ethpb.BeaconBlock{Slot: 0, ParentRoot: validGenesisRoot},
			s:             &pb.BeaconState{},
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
			fmt.Println(tt.wantErrString)
			if err.Error() != tt.wantErrString {
				t.Errorf("Store.OnBlock() error = %v, wantErr = %v", err, tt.wantErrString)
			}
		})
	}
}

func TestStore_OnAttestationErrors(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

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

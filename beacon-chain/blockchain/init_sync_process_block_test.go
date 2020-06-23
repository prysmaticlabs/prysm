package blockchain

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestGenerateState_CorrectlyGenerated(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	cfg := &Config{BeaconDB: db, StateGen: stategen.New(db, sc)}
	service, err := NewService(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	beaconState, privs := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := testutil.NewBeaconBlock()
	bodyRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	err = beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)); err != nil {
		t.Fatal(err)
	}
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	if err := beaconState.SetCurrentJustifiedCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}
	err = db.SaveBlock(context.Background(), genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	genRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	err = db.SaveState(context.Background(), beaconState, genRoot)
	if err != nil {
		t.Fatal(err)
	}

	lastBlock := &ethpb.SignedBeaconBlock{}
	for i := uint64(1); i < 10; i++ {
		block, err := testutil.GenerateFullBlock(beaconState, privs, testutil.DefaultBlockGenConfig(), i)
		if err != nil {
			t.Fatal(err)
		}
		beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
		if err != nil {
			t.Fatal(err)
		}
		err = db.SaveBlock(context.Background(), block)
		if err != nil {
			t.Fatal(err)
		}
		lastBlock = block
	}
	root, err := stateutil.BlockRoot(lastBlock.Block)
	if err != nil {
		t.Fatal(err)
	}

	newState, err := service.generateState(context.Background(), genRoot, root)
	if err != nil {
		t.Fatal(err)
	}
	if !ssz.DeepEqual(newState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		diff, _ := messagediff.PrettyDiff(newState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
		t.Errorf("Generated state is different from what is expected: %s", diff)
	}
}

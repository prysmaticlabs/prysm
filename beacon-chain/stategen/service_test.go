package stategen

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"gopkg.in/d4l3k/messagediff.v1"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestGenerateState_CorrectlyGenerated(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	beaconState, privs := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector))
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	beaconState.SetCurrentJustifiedCheckpoint(cp)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})
	err = db.SaveBlock(context.Background(), genesisBlock)
	if err != nil {
		t.Fatal(err)
	}

	genesisState := beaconState.Copy()
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

	service := New(db)
	newState, err := service.GenerateState(context.Background(), genesisState, lastBlock)
	if err != nil {
		t.Fatal(err)
	}
	if !ssz.DeepEqual(newState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		diff, _ := messagediff.PrettyDiff(newState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
		t.Errorf("Generated state is different from what is expected: %s", diff)
	}
}

package blockchain

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestSaveHead_Same(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := setupBeaconChain(t, db)

	r := [32]byte{'A'}
	service.head = &head{slot: 0, root: r}

	if err := service.saveHead(context.Background(), r); err != nil {
		t.Fatal(err)
	}

	if service.headSlot() != 0 {
		t.Error("Head did not stay the same")
	}

	if service.headRoot() != r {
		t.Error("Head did not stay the same")
	}
}

func TestSaveHead_Different(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := setupBeaconChain(t, db)

	oldRoot := [32]byte{'A'}
	service.head = &head{slot: 0, root: oldRoot}

	newHeadBlock := &ethpb.BeaconBlock{Slot: 1}
	newHeadSignedBlock := &ethpb.SignedBeaconBlock{Block: newHeadBlock}
	service.beaconDB.SaveBlock(context.Background(), newHeadSignedBlock)
	newRoot, _ := ssz.HashTreeRoot(newHeadBlock)
	headState, _ := state.InitializeFromProto(&pb.BeaconState{Slot: 1})
	service.beaconDB.SaveState(context.Background(), headState, newRoot)
	if err := service.saveHead(context.Background(), newRoot); err != nil {
		t.Fatal(err)
	}

	if service.HeadSlot() != 1 {
		t.Error("Head did not change")
	}

	cachedRoot, err := service.HeadRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cachedRoot, newRoot[:]) {
		t.Error("Head did not change")
	}
	if !reflect.DeepEqual(service.headBlock(), newHeadSignedBlock) {
		t.Error("Head did not change")
	}
	if !reflect.DeepEqual(service.headState().CloneInnerState(), headState.CloneInnerState()) {
		t.Error("Head did not change")
	}
}

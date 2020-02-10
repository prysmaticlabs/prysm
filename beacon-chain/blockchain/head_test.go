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

	service.headSlot = 0
	r := [32]byte{'A'}
	service.canonicalRoots[0] = r[:]

	if err := service.saveHead(context.Background(), r); err != nil {
		t.Fatal(err)
	}

	if service.headSlot != 0 {
		t.Error("Head did not stay the same")
	}

	cachedRoot, err := service.HeadRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cachedRoot, r[:]) {
		t.Error("Head did not stay the same")
	}
}

func TestSaveHead_Different(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := setupBeaconChain(t, db)

	service.headSlot = 0
	oldRoot := [32]byte{'A'}
	service.canonicalRoots[0] = oldRoot[:]

	newHeadBlock := &ethpb.BeaconBlock{Slot: 1}
	newHeadSignedBlock := &ethpb.SignedBeaconBlock{Block: newHeadBlock}
	service.beaconDB.SaveBlock(context.Background(), newHeadSignedBlock)
	newRoot, _ := ssz.HashTreeRoot(newHeadBlock)
	headState, _ := state.InitializeFromProto(&pb.BeaconState{})
	service.beaconDB.SaveState(context.Background(), headState, newRoot)
	if err := service.saveHead(context.Background(), newRoot); err != nil {
		t.Fatal(err)
	}

	if service.headSlot != 1 {
		t.Error("Head did not change")
	}

	cachedRoot, err := service.HeadRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cachedRoot, newRoot[:]) {
		t.Error("Head did not change")
	}
	if !reflect.DeepEqual(service.headBlock, newHeadSignedBlock) {
		t.Error("Head did not change")
	}
	if !reflect.DeepEqual(service.headState, headState) {
		t.Error("Head did not change")
	}
}

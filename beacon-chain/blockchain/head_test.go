package blockchain

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSaveHead_Same(t *testing.T) {
	db := testDB.SetupDB(t)
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
	service := setupBeaconChain(t, db)

	oldRoot := [32]byte{'A'}
	service.head = &head{slot: 0, root: oldRoot}

	newHeadBlock := &ethpb.BeaconBlock{Slot: 1}
	newHeadSignedBlock := &ethpb.SignedBeaconBlock{Block: newHeadBlock}

	if err := service.beaconDB.SaveBlock(context.Background(), newHeadSignedBlock); err != nil {
		t.Fatal(err)
	}
	newRoot, err := stateutil.BlockRoot(newHeadBlock)
	if err != nil {
		t.Fatal(err)
	}
	headState := testutil.NewBeaconState()
	if err := headState.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(context.Background(), &pb.StateSummary{Slot: 1, Root: newRoot[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(context.Background(), headState, newRoot); err != nil {
		t.Fatal(err)
	}
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

func TestSaveHead_Different_Reorg(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	service := setupBeaconChain(t, db)

	oldRoot := [32]byte{'A'}
	service.head = &head{slot: 0, root: oldRoot}

	reorgChainParent := [32]byte{'B'}
	newHeadBlock := &ethpb.BeaconBlock{
		Slot:       1,
		ParentRoot: reorgChainParent[:],
	}
	newHeadSignedBlock := &ethpb.SignedBeaconBlock{Block: newHeadBlock}

	if err := service.beaconDB.SaveBlock(context.Background(), newHeadSignedBlock); err != nil {
		t.Fatal(err)
	}
	newRoot, err := stateutil.BlockRoot(newHeadBlock)
	if err != nil {
		t.Fatal(err)
	}
	headState := testutil.NewBeaconState()
	if err := headState.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(context.Background(), &pb.StateSummary{Slot: 1, Root: newRoot[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(context.Background(), headState, newRoot); err != nil {
		t.Fatal(err)
	}
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
	testutil.AssertLogsContain(t, hook, "Chain reorg occurred")
}

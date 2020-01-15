package blockchain

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
)

func TestHeadSlot_DataRace(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	s := &Service{
		beaconDB:       db,
		canonicalRoots: make(map[uint64][]byte),
	}
	go func() {
		s.saveHead(
			context.Background(),
			&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 777}},
			[32]byte{},
		)
	}()
	s.HeadSlot()
}

func TestHeadRoot_DataRace(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	s := &Service{
		beaconDB:       db,
		canonicalRoots: make(map[uint64][]byte),
	}
	go func() {
		s.saveHead(
			context.Background(),
			&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 777}},
			[32]byte{},
		)
	}()
	if _, err := s.HeadRoot(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestHeadBlock_DataRace(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	s := &Service{
		beaconDB:       db,
		canonicalRoots: make(map[uint64][]byte),
	}
	go func() {
		s.saveHead(
			context.Background(),
			&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 777}},
			[32]byte{},
		)
	}()
	s.HeadBlock()
}

func TestHeadState_DataRace(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	s := &Service{
		beaconDB:       db,
		canonicalRoots: make(map[uint64][]byte),
	}
	go func() {
		s.saveHead(
			context.Background(),
			&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 777}},
			[32]byte{},
		)
	}()
	s.HeadState(context.Background())
}

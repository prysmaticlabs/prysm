package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

func TestAttestHead(t *testing.T) {
	mockOperationService := &mockOperationService{}
	attesterServer := &AttesterServer{
		operationService: mockOperationService,
	}
	req := &pbp2p.Attestation{
		Data: &pbp2p.AttestationData{
			Slot:                 999,
			Shard:                1,
			ShardBlockRootHash32: []byte{'a'},
		},
	}
	if _, err := attesterServer.AttestHead(context.Background(), req); err != nil {
		t.Errorf("Could not attest head correctly: %v", err)
	}
}

func TestAttestationInfoAtSlot_NilHeadAtSlot(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	attesterServer := &AttesterServer{
		beaconDB: db,
	}
	want := "no block found at slot 5"
	req := &pb.AttestationInfoRequest{
		Slot: 5,
	}
	if _, err := attesterServer.AttestationInfoAtSlot(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestAttestationInfoAtSlot_EpochBoundaryRootFailure(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	beaconState := &pbp2p.BeaconState{
		Slot:                   5,
		LatestBlockRootHash32S: make([][]byte, 20),
	}
	block := blocks.NewGenesisBlock([]byte("stateroot"))
	block.Slot = 100
	attesterServer := &AttesterServer{
		beaconDB: db,
	}
	if err := attesterServer.beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(block, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	want := "could not get epoch boundary block"
	req := &pb.AttestationInfoRequest{
		Slot: 100,
	}
	if _, err := attesterServer.AttestationInfoAtSlot(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

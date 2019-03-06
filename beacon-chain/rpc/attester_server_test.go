package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestAttestHead_OK(t *testing.T) {
	mockOperationService := &mockOperationService{}
	attesterServer := &AttesterServer{
		operationService: mockOperationService,
	}
	req := &pbp2p.Attestation{
		Data: &pbp2p.AttestationData{
			Slot:                    999,
			Shard:                   1,
			CrosslinkDataRootHash32: []byte{'a'},
		},
	}
	if _, err := attesterServer.AttestHead(context.Background(), req); err != nil {
		t.Errorf("Could not attest head correctly: %v", err)
	}
}

func TestAttestationDataAtSlot_EpochBoundaryFailure(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	beaconState := &pbp2p.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + 3*params.BeaconConfig().SlotsPerEpoch,
		LatestBlockRootHash32S: make([][]byte, 20),
		JustifiedEpoch:         params.BeaconConfig().GenesisEpoch + 1*params.BeaconConfig().GenesisEpoch,
	}
	block := blocks.NewGenesisBlock([]byte("stateroot"))
	block.Slot = params.BeaconConfig().GenesisSlot + 3*params.BeaconConfig().SlotsPerEpoch + 1
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
	req := &pb.AttestationDataRequest{}
	if _, err := attesterServer.AttestationDataAtSlot(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestAttestationDataAtSlot_JustifiedBlockFailure(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	beaconState := &pbp2p.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch + 2,
		LatestBlockRootHash32S: make([][]byte, params.BeaconConfig().LatestBlockRootsLength),
	}
	block := &pbp2p.BeaconBlock{
		Slot: params.BeaconConfig().GenesisSlot + 1,
	}
	attesterServer := &AttesterServer{
		beaconDB: db,
	}
	if err := attesterServer.beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(block, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	epochBoundaryBlock := &pbp2p.BeaconBlock{
		Slot: params.BeaconConfig().GenesisSlot + 1,
	}
	if err := attesterServer.beaconDB.SaveBlock(epochBoundaryBlock); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(epochBoundaryBlock, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	want := "could not get justified block"
	req := &pb.AttestationDataRequest{}
	if _, err := attesterServer.AttestationDataAtSlot(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestAttestationDataAtSlot_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	block := &pbp2p.BeaconBlock{
		Slot: 1 + params.BeaconConfig().GenesisSlot,
	}
	epochBoundaryBlock := &pbp2p.BeaconBlock{
		Slot: 1*params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot,
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: 2*params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot,
	}
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	justifiedBlockRoot, err := hashutil.HashBeaconBlock(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	epochBoundaryRoot, err := hashutil.HashBeaconBlock(epochBoundaryBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	beaconState := &pbp2p.BeaconState{
		Slot:                   3*params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot + 1,
		JustifiedEpoch:         2 + params.BeaconConfig().GenesisEpoch,
		LatestBlockRootHash32S: make([][]byte, params.BeaconConfig().LatestBlockRootsLength),
		LatestCrosslinks: []*pbp2p.Crosslink{
			{
				CrosslinkDataRootHash32: []byte("A"),
			},
		},
	}
	beaconState.LatestBlockRootHash32S[1] = blockRoot[:]
	beaconState.LatestBlockRootHash32S[1*params.BeaconConfig().SlotsPerEpoch] = epochBoundaryRoot[:]
	beaconState.LatestBlockRootHash32S[2*params.BeaconConfig().SlotsPerEpoch] = justifiedBlockRoot[:]
	attesterServer := &AttesterServer{
		beaconDB: db,
	}
	if err := attesterServer.beaconDB.SaveBlock(epochBoundaryBlock); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(epochBoundaryBlock, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	if err := attesterServer.beaconDB.SaveBlock(justifiedBlock); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(justifiedBlock, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	if err := attesterServer.beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(block, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	req := &pb.AttestationDataRequest{
		Shard: 0,
	}
	res, err := attesterServer.AttestationDataAtSlot(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get attestation info at slot: %v", err)
	}
	expectedInfo := &pb.AttestationDataResponse{
		BeaconBlockRootHash32:    blockRoot[:],
		JustifiedEpoch:           2 + params.BeaconConfig().GenesisEpoch,
		JustifiedBlockRootHash32: justifiedBlockRoot[:],
		LatestCrosslink: &pbp2p.Crosslink{
			CrosslinkDataRootHash32: []byte("A"),
		},
	}

	if !proto.Equal(res, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

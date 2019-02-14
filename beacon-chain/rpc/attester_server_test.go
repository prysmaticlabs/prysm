package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/gogo/protobuf/proto"
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

func TestAttestationInfoAtSlot_EpochBoundaryFailure(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	beaconState := &pbp2p.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + 3*params.BeaconConfig().GenesisEpoch,
		LatestBlockRootHash32S: make([][]byte, 20),
		JustifiedEpoch:         params.BeaconConfig().GenesisEpoch,
	}
	block := blocks.NewGenesisBlock([]byte("stateroot"))
	block.Slot = params.BeaconConfig().GenesisSlot
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
	req := &pb.AttestationInfoRequest{}
	if _, err := attesterServer.AttestationInfoAtSlot(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestAttestationInfoAtSlot_JustifiedBlockFailure(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	beaconState := &pbp2p.BeaconState{
		Slot:                   params.BeaconConfig().EpochLength + 2,
		LatestBlockRootHash32S: make([][]byte, 20),
	}
	block := &pbp2p.BeaconBlock{
		Slot: 1,
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
		Slot: params.BeaconConfig().EpochLength + 1,
	}
	if err := attesterServer.beaconDB.SaveBlock(epochBoundaryBlock); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(epochBoundaryBlock, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	want := "could not get justified block"
	req := &pb.AttestationInfoRequest{}
	if _, err := attesterServer.AttestationInfoAtSlot(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestAttestationInfoAtSlot_Ok(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	block := &pbp2p.BeaconBlock{
		Slot: params.BeaconConfig().GenesisSlot + 3*params.BeaconConfig().EpochLength + 1,
	}
	epochBoundarySlot := params.BeaconConfig().GenesisSlot + 3*params.BeaconConfig().EpochLength
	epochBoundaryBlock := &pbp2p.BeaconBlock{
		Slot: epochBoundarySlot,
	}
	justifiedSlot := epochBoundarySlot
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: justifiedSlot,
	}
	blockRoot, err := ssz.TreeHash(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	justifiedBlockRoot, err := ssz.TreeHash(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	epochBoundaryRoot, err := ssz.TreeHash(epochBoundaryBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	beaconState := &pbp2p.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + 3*params.BeaconConfig().EpochLength + 1,
		JustifiedEpoch:         epochBoundarySlot / params.BeaconConfig().EpochLength,
		LatestBlockRootHash32S: make([][]byte, 2),
		LatestCrosslinks: []*pbp2p.Crosslink{
			{
				ShardBlockRootHash32: []byte("A"),
			},
		},
	}
	beaconState.LatestBlockRootHash32S[epochBoundarySlot%2] = epochBoundaryRoot[:]
	beaconState.LatestBlockRootHash32S[justifiedSlot%2] = justifiedBlockRoot[:]
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
	req := &pb.AttestationInfoRequest{
		Shard: 0,
	}
	res, err := attesterServer.AttestationInfoAtSlot(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get attestation info at slot: %v", err)
	}
	expectedInfo := &pb.AttestationInfoResponse{
		BeaconBlockRootHash32:    blockRoot[:],
		EpochBoundaryRootHash32:  epochBoundaryRoot[:],
		JustifiedEpoch:           justifiedSlot / params.BeaconConfig().EpochLength,
		JustifiedBlockRootHash32: justifiedBlockRoot[:],
		LatestCrosslink: &pbp2p.Crosslink{
			ShardBlockRootHash32: []byte("A"),
		},
	}
	if !proto.Equal(res, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

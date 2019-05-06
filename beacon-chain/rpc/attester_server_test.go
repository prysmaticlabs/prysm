package rpc

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type mockBroadcaster struct{}

func (m *mockBroadcaster) Broadcast(ctx context.Context, msg proto.Message) {
}

func TestAttestHead_OK(t *testing.T) {
	mockOperationService := &mockOperationService{}
	attesterServer := &AttesterServer{
		operationService: mockOperationService,
		p2p:              &mockBroadcaster{},
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

func TestAttestationDataAtSlot_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

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
		JustifiedRoot: justifiedBlockRoot[:],
	}
	beaconState.LatestBlockRootHash32S[1] = blockRoot[:]
	beaconState.LatestBlockRootHash32S[1*params.BeaconConfig().SlotsPerEpoch] = epochBoundaryRoot[:]
	beaconState.LatestBlockRootHash32S[2*params.BeaconConfig().SlotsPerEpoch] = justifiedBlockRoot[:]
	attesterServer := &AttesterServer{
		beaconDB: db,
		p2p:      &mockBroadcaster{},
	}
	if err := attesterServer.beaconDB.SaveBlock(epochBoundaryBlock); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(ctx, epochBoundaryBlock, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	if err := attesterServer.beaconDB.SaveBlock(justifiedBlock); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(ctx, justifiedBlock, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	if err := attesterServer.beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(ctx, block, beaconState); err != nil {
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
		HeadSlot:                 beaconState.Slot,
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

func TestAttestationDataAtSlot_handlesFarAwayJustifiedEpoch(t *testing.T) {
	// Scenario:
	//
	// State slot = 10000
	// Last justified slot = epoch start of 1500
	// LatestBlockRootsLength = 8192
	//
	// More background: https://github.com/prysmaticlabs/prysm/issues/2153
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	// Ensure LatestBlockRootsLength matches scenario
	cfg := params.BeaconConfig()
	cfg.LatestBlockRootsLength = 8192
	params.OverrideBeaconConfig(cfg)

	block := &pbp2p.BeaconBlock{
		Slot: 10000 + params.BeaconConfig().GenesisSlot,
	}
	epochBoundaryBlock := &pbp2p.BeaconBlock{
		Slot: helpers.StartSlot(helpers.SlotToEpoch(10000 + params.BeaconConfig().GenesisSlot)),
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: helpers.StartSlot(helpers.SlotToEpoch(1500+params.BeaconConfig().GenesisSlot)) - 2, // Imagine two skip block
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
		Slot:                   10000 + params.BeaconConfig().GenesisSlot,
		JustifiedEpoch:         helpers.SlotToEpoch(1500 + params.BeaconConfig().GenesisSlot),
		LatestBlockRootHash32S: make([][]byte, params.BeaconConfig().LatestBlockRootsLength),
		LatestCrosslinks: []*pbp2p.Crosslink{
			{
				CrosslinkDataRootHash32: []byte("A"),
			},
		},
		JustifiedRoot: justifiedBlockRoot[:],
	}
	beaconState.LatestBlockRootHash32S[1] = blockRoot[:]
	beaconState.LatestBlockRootHash32S[1*params.BeaconConfig().SlotsPerEpoch] = epochBoundaryRoot[:]
	beaconState.LatestBlockRootHash32S[2*params.BeaconConfig().SlotsPerEpoch] = justifiedBlockRoot[:]
	attesterServer := &AttesterServer{
		beaconDB: db,
		p2p:      &mockBroadcaster{},
	}
	if err := attesterServer.beaconDB.SaveBlock(epochBoundaryBlock); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(ctx, epochBoundaryBlock, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	if err := attesterServer.beaconDB.SaveBlock(justifiedBlock); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(ctx, justifiedBlock, beaconState); err != nil {
		t.Fatalf("Could not update chain head in test db: %v", err)
	}
	if err := attesterServer.beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(ctx, block, beaconState); err != nil {
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
		HeadSlot:                 10000 + params.BeaconConfig().GenesisSlot,
		BeaconBlockRootHash32:    blockRoot[:],
		JustifiedEpoch:           helpers.SlotToEpoch(1500 + params.BeaconConfig().GenesisSlot),
		JustifiedBlockRootHash32: justifiedBlockRoot[:],
		LatestCrosslink: &pbp2p.Crosslink{
			CrosslinkDataRootHash32: []byte("A"),
		},
	}

	if !proto.Equal(res, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

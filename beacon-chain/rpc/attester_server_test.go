package rpc

import (
	"context"
	"sync"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type mockBroadcaster struct{}

func (m *mockBroadcaster) Broadcast(ctx context.Context, msg proto.Message) {
}

func TestSubmitAttestation_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	mockOperationService := &mockOperationService{}
	attesterServer := &AttesterServer{
		operationService: mockOperationService,
		p2p:              &mockBroadcaster{},
		beaconDB:         db,
		cache:            cache.NewAttestationCache(),
	}
	head := &pbp2p.BeaconBlock{
		Slot:       999,
		ParentRoot: []byte{'a'},
	}
	if err := attesterServer.beaconDB.SaveBlock(head); err != nil {
		t.Fatal(err)
	}
	root, err := ssz.SigningRoot(head)
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*pbp2p.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/16)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pbp2p.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}

	state := &pbp2p.BeaconState{
		Slot:             params.BeaconConfig().SlotsPerEpoch + 1,
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	if err := db.SaveState(context.Background(), state); err != nil {
		t.Fatal(err)
	}

	req := &pbp2p.Attestation{
		Data: &pbp2p.AttestationData{
			BeaconBlockRoot: root[:],
			Crosslink: &pbp2p.Crosslink{
				Shard:    935,
				DataRoot: []byte{'a'},
			},
			Source: &pbp2p.Checkpoint{},
			Target: &pbp2p.Checkpoint{},
		},
	}
	if _, err := attesterServer.SubmitAttestation(context.Background(), req); err != nil {
		t.Errorf("Could not attest head correctly: %v", err)
	}
}

func TestRequestAttestation_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	block := &pbp2p.BeaconBlock{
		Slot: 3*params.BeaconConfig().SlotsPerEpoch + 1,
	}
	targetBlock := &pbp2p.BeaconBlock{
		Slot: 1 * params.BeaconConfig().SlotsPerEpoch,
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: 2 * params.BeaconConfig().SlotsPerEpoch,
	}
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	justifiedRoot, err := ssz.SigningRoot(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for justified block: %v", err)
	}
	targetRoot, err := ssz.SigningRoot(targetBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for target block: %v", err)
	}

	beaconState := &pbp2p.BeaconState{
		Slot:       3*params.BeaconConfig().SlotsPerEpoch + 1,
		BlockRoots: make([][]byte, params.BeaconConfig().HistoricalRootsLimit),
		CurrentCrosslinks: []*pbp2p.Crosslink{
			{
				DataRoot: []byte("A"),
			},
		},
		PreviousCrosslinks: []*pbp2p.Crosslink{
			{
				DataRoot: []byte("A"),
			},
		},
		CurrentJustifiedCheckpoint: &pbp2p.Checkpoint{
			Epoch: 2,
			Root:  justifiedRoot[:],
		},
	}
	beaconState.BlockRoots[1] = blockRoot[:]
	beaconState.BlockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
	beaconState.BlockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
	attesterServer := &AttesterServer{
		beaconDB: db,
		p2p:      &mockBroadcaster{},
		cache:    cache.NewAttestationCache(),
	}
	if err := attesterServer.beaconDB.SaveBlock(targetBlock); err != nil {
		t.Fatalf("Could not save block in test db: %v", err)
	}
	if err := attesterServer.beaconDB.UpdateChainHead(ctx, targetBlock, beaconState); err != nil {
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
	req := &pb.AttestationRequest{
		Shard: 0,
		Slot:  3*params.BeaconConfig().SlotsPerEpoch + 1,
	}
	res, err := attesterServer.RequestAttestation(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get attestation info at slot: %v", err)
	}

	crosslinkRoot, err := ssz.HashTreeRoot(beaconState.CurrentCrosslinks[req.Shard])
	if err != nil {
		t.Fatal(err)
	}

	expectedInfo := &pbp2p.AttestationData{
		BeaconBlockRoot: blockRoot[:],
		Source: &pbp2p.Checkpoint{
			Epoch: 2,
			Root:  justifiedRoot[:],
		},
		Target: &pbp2p.Checkpoint{
			Epoch: 3,
		},
		Crosslink: &pbp2p.Crosslink{
			EndEpoch:   3,
			ParentRoot: crosslinkRoot[:],
			DataRoot:   params.BeaconConfig().ZeroHash[:],
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
	// HistoricalRootsLimit = 8192
	//
	// More background: https://github.com/prysmaticlabs/prysm/issues/2153
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	// Ensure HistoricalRootsLimit matches scenario
	cfg := params.BeaconConfig()
	cfg.HistoricalRootsLimit = 8192
	params.OverrideBeaconConfig(cfg)

	block := &pbp2p.BeaconBlock{
		Slot: 10000,
	}
	epochBoundaryBlock := &pbp2p.BeaconBlock{
		Slot: helpers.StartSlot(helpers.SlotToEpoch(10000)),
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: helpers.StartSlot(helpers.SlotToEpoch(1500)) - 2, // Imagine two skip block
	}
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	justifiedBlockRoot, err := ssz.SigningRoot(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	epochBoundaryRoot, err := ssz.SigningRoot(epochBoundaryBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	beaconState := &pbp2p.BeaconState{
		Slot:       10000,
		BlockRoots: make([][]byte, params.BeaconConfig().HistoricalRootsLimit),
		PreviousCrosslinks: []*pbp2p.Crosslink{
			{
				DataRoot: []byte("A"),
			},
		},
		CurrentCrosslinks: []*pbp2p.Crosslink{
			{
				DataRoot: []byte("A"),
			},
		},
		CurrentJustifiedCheckpoint: &pbp2p.Checkpoint{
			Epoch: helpers.SlotToEpoch(1500),
			Root:  justifiedBlockRoot[:],
		},
	}
	beaconState.BlockRoots[1] = blockRoot[:]
	beaconState.BlockRoots[1*params.BeaconConfig().SlotsPerEpoch] = epochBoundaryRoot[:]
	beaconState.BlockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedBlockRoot[:]
	attesterServer := &AttesterServer{
		beaconDB: db,
		p2p:      &mockBroadcaster{},
		cache:    cache.NewAttestationCache(),
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
	req := &pb.AttestationRequest{
		Shard: 0,
		Slot:  10000,
	}
	res, err := attesterServer.RequestAttestation(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get attestation info at slot: %v", err)
	}

	crosslinkRoot, err := ssz.HashTreeRoot(beaconState.CurrentCrosslinks[req.Shard])
	if err != nil {
		t.Fatal(err)
	}

	expectedInfo := &pbp2p.AttestationData{
		BeaconBlockRoot: blockRoot[:],
		Source: &pbp2p.Checkpoint{
			Epoch: helpers.SlotToEpoch(1500),
			Root:  justifiedBlockRoot[:],
		},
		Target: &pbp2p.Checkpoint{
			Epoch: 156,
		},
		Crosslink: &pbp2p.Crosslink{
			ParentRoot: crosslinkRoot[:],
			EndEpoch:   params.BeaconConfig().SlotsPerEpoch,
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		},
	}

	if !proto.Equal(res, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

func TestAttestationDataAtSlot_handlesInProgressRequest(t *testing.T) {
	ctx := context.Background()
	server := &AttesterServer{
		cache: cache.NewAttestationCache(),
	}

	req := &pb.AttestationRequest{
		Shard: 1,
		Slot:  2,
	}

	res := &pbp2p.AttestationData{
		Target: &pbp2p.Checkpoint{Epoch: 55},
	}

	if err := server.cache.MarkInProgress(req); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		response, err := server.RequestAttestation(ctx, req)
		if err != nil {
			t.Error(err)
		}
		if !proto.Equal(res, response) {
			t.Error("Expected  equal responses from cache")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := server.cache.Put(ctx, req, res); err != nil {
			t.Error(err)
		}
		if err := server.cache.MarkNotInProgress(req); err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()
}

package sync

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"

	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

var topicMappings = map[pb.Topic]proto.Message{
	pb.Topic_BEACON_BLOCK_ANNOUNCE:               &pb.BeaconBlockAnnounce{},
	pb.Topic_BEACON_BLOCK_REQUEST:                &pb.BeaconBlockRequest{},
	pb.Topic_BEACON_BLOCK_REQUEST_BY_SLOT_NUMBER: &pb.BeaconBlockRequestBySlotNumber{},
	pb.Topic_BEACON_BLOCK_RESPONSE:               &pb.BeaconBlockResponse{},
	pb.Topic_BATCHED_BEACON_BLOCK_REQUEST:        &pb.BatchedBeaconBlockRequest{},
	pb.Topic_BATCHED_BEACON_BLOCK_RESPONSE:       &pb.BatchedBeaconBlockResponse{},
	pb.Topic_CHAIN_HEAD_REQUEST:                  &pb.ChainHeadRequest{},
	pb.Topic_CHAIN_HEAD_RESPONSE:                 &pb.ChainHeadResponse{},
	pb.Topic_BEACON_STATE_HASH_ANNOUNCE:          &pb.BeaconStateHashAnnounce{},
	pb.Topic_BEACON_STATE_REQUEST:                &pb.BeaconStateRequest{},
	pb.Topic_BEACON_STATE_RESPONSE:               &pb.BeaconStateResponse{},
}

func setUpSyncedService(numOfBlocks int, t *testing.T) (*Service, *db.BeaconDB, *p2p.Server) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	privKeys, err := bd.SetupBackend(100)
	if err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}

	beacondb, err := db.SetupDB()
	if err != nil {
		t.Fatalf("Could not setup beacon db %v", err)
	}

	if err := beacondb.SaveState(bd.State()); err != nil {
		t.Fatalf("Could not save state %v", err)
	}

	memBlocks := bd.InMemoryBlocks()
	if err := beacondb.SaveBlock(memBlocks[0]); err != nil {
		t.Fatalf("Could not save block %v", err)
	}

	if err := beacondb.UpdateChainHead(memBlocks[0], bd.State()); err != nil {
		t.Fatalf("Could not update chain head %v", err)
	}

	mockPow := &genesisPowChain{
		feed: new(event.Feed),
	}

	mockChain := &mockChainService{
		feed: new(event.Feed),
	}

	mockServer, err := p2p.MockServer(t)
	if err != nil {
		t.Fatalf("Could not create p2p server %v", err)
	}

	for k, v := range topicMappings {
		mockServer.RegisterTopic(k.String(), v)
	}

	cfg := &Config{
		ChainService:     mockChain,
		BeaconDB:         beacondb,
		OperationService: &mockOperationService{},
		P2P:              mockServer,
		PowChainService:  mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	mockServer.Start()
	go ss.run()
	for !ss.Querier.chainStarted {
		mockPow.feed.Send(time.Now())
	}

	for i := 0; i < numOfBlocks; i++ {
		bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{}, privKeys)
		blocks := bd.InMemoryBlocks()
		beacondb.SaveBlock(blocks[i+1])
	}

	memBlocks = bd.InMemoryBlocks()

	if err := beacondb.UpdateChainHead(memBlocks[len(memBlocks)-1], bd.State()); err != nil {
		t.Fatalf("Could not update chain head %v", err)
	}

	return ss, beacondb, mockServer
}

func setUpUnSyncedService(t *testing.T) (*Service, *db.BeaconDB, *p2p.Server) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	if _, err := bd.SetupBackend(100); err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}

	beacondb, err := db.SetupDB()
	if err != nil {
		t.Fatalf("Could not setup beacon db %v", err)
	}

	if err := beacondb.SaveState(bd.State()); err != nil {
		t.Fatalf("Could not save state %v", err)
	}

	memBlocks := bd.InMemoryBlocks()

	if err := beacondb.SaveBlock(memBlocks[0]); err != nil {
		t.Fatalf("Could not save block %v", err)
	}

	if err := beacondb.UpdateChainHead(memBlocks[0], bd.State()); err != nil {
		t.Fatalf("Could not update chain head %v", err)
	}

	mockPow := &afterGenesisPowChain{
		feed: new(event.Feed),
	}

	mockChain := &mockChainService{
		feed: new(event.Feed),
	}

	mockServer, err := p2p.MockServer(t)
	if err != nil {
		t.Fatalf("Could not create p2p server %v", err)
	}

	for k, v := range topicMappings {
		mockServer.RegisterTopic(k.String(), v)
	}

	cfg := &Config{
		ChainService:     mockChain,
		BeaconDB:         beacondb,
		OperationService: &mockOperationService{},
		P2P:              mockServer,
		PowChainService:  mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	mockServer.Start()
	go ss.run()

	for ss.Querier.curentHeadSlot == 0 {
		mockServer.Feed(&pb.ChainHeadResponse{}).Send(p2p.Message{
			Ctx: context.Background(),
			Data: &pb.ChainHeadResponse{
				Slot: params.BeaconConfig().GenesisSlot + 10,
				Hash: []byte{'t', 'e', 's', 't'},
			},
		})
	}

	return ss, beacondb, mockServer
}

func TestServices(t *testing.T) {
	ss, syncedDB, sp2p := setUpSyncedService(10, t)
	defer ss.Stop()
	defer db.TeardownDB(syncedDB)
	defer sp2p.Stop()

	us, unSyncedDB, up2p := setUpUnSyncedService(t)
	defer us.Stop()
	defer db.TeardownDB(unSyncedDB)
	defer up2p.Stop()

	for us.Querier.curentHeadSlot != 0 {

	}
}

func TestSetupTestingEnvironment(t *testing.T) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	privKeys, err := bd.SetupBackend(100)
	if err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}

	beacondb, err := db.SetupDB()
	if err != nil {
		t.Fatalf("Could not setup beacon db %v", err)
	}
	defer db.TeardownDB(beacondb)

	if err := beacondb.SaveState(bd.State()); err != nil {
		t.Fatalf("Could not save state %v", err)
	}

	mockPow := &genesisPowChain{
		feed: new(event.Feed),
	}

	mockChain := &mockChainService{
		feed: new(event.Feed),
	}

	mockServer, err := p2p.MockServer(t)
	if err != nil {
		t.Fatalf("Could not create p2p server %v", err)
	}

	cfg := &Config{
		ChainService:     mockChain,
		BeaconDB:         beacondb,
		OperationService: &mockOperationService{},
		P2P:              mockServer,
		PowChainService:  mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	go ss.run()
	for !ss.Querier.chainStarted {
		mockPow.feed.Send(time.Now())
	}

	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{}, privKeys)
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{}, privKeys)
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{}, privKeys)
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{}, privKeys)
	ctx := context.Background()
	blocks := bd.InMemoryBlocks()
	newChan := make(chan *pb.BeaconBlock, 5)
	sub := mockChain.feed.Subscribe(newChan)
	defer sub.Unsubscribe()

	go func() {
		for {
			select {
			case <-sub.Err():
				return
			case blk := <-newChan:
				beacondb.SaveBlock(blk)
				t.Log(blk)
			}
		}
	}()

	for i := 0; i < 4; i++ {
		mockServer.Feed(&pb.BeaconBlockResponse{}).Send(p2p.Message{
			Ctx: ctx,
			Data: &pb.BeaconBlockResponse{
				Block: blocks[i],
			}})
	}

	ss.Stop()
	bd.Shutdown()
}

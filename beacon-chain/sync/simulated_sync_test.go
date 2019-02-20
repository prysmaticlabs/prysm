package sync

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

type genesisPowChain struct {
	feed *event.Feed
}

func (mp *genesisPowChain) HasChainStartLogOccurred() (bool, uint64, error) {
	return false, 0, nil
}

func (mp *genesisPowChain) ChainStartFeed() *event.Feed {
	return mp.feed
}

type afterGenesisPowChain struct {
	feed *event.Feed
}

func (mp *afterGenesisPowChain) HasChainStartLogOccurred() (bool, uint64, error) {
	return true, 0, nil
}

func (mp *afterGenesisPowChain) ChainStartFeed() *event.Feed {
	return mp.feed
}

func setUpSyncedService(numOfBlocks int, t *testing.T) (*Service, *db.BeaconDB) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	if err := bd.SetupBackend(100); err != nil {
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
		Powchain:         mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	go ss.run()
	for !ss.Querier.isChainStart {
		mockPow.feed.Send(time.Now())
	}

	for i := 0; i < numOfBlocks; i++ {
		bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
		blocks := bd.InMemoryBlocks()
		beacondb.SaveBlock(blocks[i])
	}

	return ss, beacondb
}

func setUpUnSyncedService(numOfBlocks int, t *testing.T) (*Service, *db.BeaconDB) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	if err := bd.SetupBackend(100); err != nil {
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

	cfg := &Config{
		ChainService:     mockChain,
		BeaconDB:         beacondb,
		OperationService: &mockOperationService{},
		P2P:              mockServer,
		Powchain:         mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	go ss.run()

	ss.Querier.responseBuf <- p2p.Message{
		Ctx: context.Background(),
		Data: &pb.ChainHeadResponse{
			Slot: 10,
			Hash: []byte{'t', 'e', 's', 't'},
		},
	}

	for len(ss.Querier.responseBuf) != 0 {

	}

	return ss, beacondb
}

func TestServices(t *testing.T) {
	setUpUnSyncedService(10, t)
}

func TestSetupTestingEnvironment(t *testing.T) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	if err := bd.SetupBackend(100); err != nil {
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
		Powchain:         mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	go ss.run()
	for !ss.Querier.isChainStart {
		mockPow.feed.Send(time.Now())
	}

	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
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

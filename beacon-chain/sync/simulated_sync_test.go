package sync

import (
	"context"
	"reflect"
	"sync"
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

type simulatedP2P struct {
	subsChannels map[reflect.Type]*event.Feed
	mutex        *sync.RWMutex
	ctx          context.Context
}

func (sim *simulatedP2P) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()

	protoType := reflect.TypeOf(msg)

	feed, ok := sim.subsChannels[protoType]
	if !ok {
		nFeed := new(event.Feed)
		sim.subsChannels[protoType] = nFeed
		return nFeed.Subscribe(channel)
	}
	return feed.Subscribe(channel)
}

func (sim *simulatedP2P) Broadcast(msg proto.Message) {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()

	protoType := reflect.TypeOf(msg)

	feed, ok := sim.subsChannels[protoType]
	if !ok {
		return
	}

	feed.Send(p2p.Message{Ctx: sim.ctx, Data: msg})
}

func (sim *simulatedP2P) Send(msg proto.Message, peer p2p.Peer) {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()

	protoType := reflect.TypeOf(msg)

	feed, ok := sim.subsChannels[protoType]
	if !ok {
		return
	}

	feed.Send(p2p.Message{Ctx: sim.ctx, Data: msg})
}

func setUpSyncedService(numOfBlocks int, simP2P *simulatedP2P, t *testing.T) (*Service, *db.BeaconDB) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	privKeys, err := bd.SetupBackend(100)
	if err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}
	defer db.TeardownDB(bd.BeaconDB)

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

	cfg := &Config{
		ChainService:     mockChain,
		BeaconDB:         beacondb,
		OperationService: &mockOperationService{},
		P2P:              simP2P,
		PowChainService:  mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	go ss.run()
	for !ss.Querier.chainStarted {
		mockPow.feed.Send(time.Now())
	}

	for i := 1; i <= numOfBlocks; i++ {
		if err := bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{}, privKeys); err != nil {
			t.Fatalf("Unable to generate block in simulated backend %v", err)
		}
		blocks := bd.InMemoryBlocks()
		if err := beacondb.SaveBlock(blocks[i]); err != nil {
			t.Fatalf("Unable to save block %v", err)
		}
		if err := beacondb.UpdateChainHead(blocks[i], bd.State()); err != nil {
			t.Fatalf("Unable to update chain head %v", err)
		}
	}

	return ss, beacondb
}

func setUpUnSyncedService(simP2P *simulatedP2P, t *testing.T) (*Service, *db.BeaconDB) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	if _, err := bd.SetupBackend(100); err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}
	defer db.TeardownDB(bd.BeaconDB)

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

	cfg := &Config{
		ChainService:     mockChain,
		BeaconDB:         beacondb,
		OperationService: &mockOperationService{},
		P2P:              simP2P,
		PowChainService:  mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	go ss.run()

	for ss.Querier.curentHeadSlot == 0 {
		simP2P.Send(&pb.ChainHeadResponse{
			Slot: params.BeaconConfig().GenesisSlot + 10,
			Hash: []byte{'t', 'e', 's', 't'},
		}, p2p.Peer{})
	}

	return ss, beacondb
}

func TestServices(t *testing.T) {
	numOfBlocks := 10
	newP2P := &simulatedP2P{
		subsChannels: make(map[reflect.Type]*event.Feed),
		mutex:        new(sync.RWMutex),
		ctx:          context.Background(),
	}
	ss, syncedDB := setUpSyncedService(numOfBlocks, newP2P, t)
	defer ss.Stop()
	defer db.TeardownDB(syncedDB)

	us, unSyncedDB := setUpUnSyncedService(newP2P, t)
	defer us.Stop()
	defer db.TeardownDB(unSyncedDB)

	syncedChan := make(chan uint64)

	sub := us.InitialSync.SyncedFeed().Subscribe(syncedChan)
	defer sub.Unsubscribe()

	highestSlot := <-syncedChan

	if highestSlot != uint64(numOfBlocks)+params.BeaconConfig().GenesisSlot {
		t.Errorf("Sync services didnt sync to expectecd slot, expected %d but got %d",
			uint64(numOfBlocks)+params.BeaconConfig().GenesisSlot, highestSlot)
	}
}

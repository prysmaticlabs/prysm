package sync

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
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

func (sim *simulatedP2P) Broadcast(_ context.Context, msg proto.Message) {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()

	protoType := reflect.TypeOf(msg)

	feed, ok := sim.subsChannels[protoType]
	if !ok {
		return
	}

	feed.Send(p2p.Message{Ctx: sim.ctx, Data: msg})
}

func (sim *simulatedP2P) Send(ctx context.Context, msg proto.Message, peerID peer.ID) error {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()

	protoType := reflect.TypeOf(msg)

	feed, ok := sim.subsChannels[protoType]
	if !ok {
		return nil
	}

	feed.Send(p2p.Message{Ctx: sim.ctx, Data: msg})
	return nil
}

func setupSimBackendAndDB(t *testing.T) (*backend.SimulatedBackend, *db.BeaconDB, []*bls.SecretKey) {
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

	return bd, beacondb, privKeys
}

func setUpSyncedService(numOfBlocks int, simP2P *simulatedP2P, t *testing.T) (*Service, *db.BeaconDB) {
	bd, beacondb, privKeys := setupSimBackendAndDB(t)
	defer bd.Shutdown()
	defer db.TeardownDB(bd.DB())

	mockPow := &genesisPowChain{
		feed: new(event.Feed),
	}

	mockChain := &mockChainService{
		bFeed: new(event.Feed),
		sFeed: new(event.Feed),
		cFeed: new(event.Feed),
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
		mockChain.sFeed.Send(time.Now())
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

func setUpUnSyncedService(simP2P *simulatedP2P, stateRoot [32]byte, t *testing.T) (*Service, *db.BeaconDB) {
	bd, beacondb, privKeys := setupSimBackendAndDB(t)
	defer bd.Shutdown()
	defer db.TeardownDB(bd.DB())

	mockPow := &afterGenesisPowChain{
		feed: new(event.Feed),
	}

	mockChain := &mockChainService{
		bFeed: new(event.Feed),
		sFeed: new(event.Feed),
		cFeed: new(event.Feed),
	}

	// we add in 2 blocks to the unsynced node so that, we dont request the beacon state from the
	// synced node to reduce test time.
	for i := 1; i <= 2; i++ {
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

	cfg := &Config{
		ChainService:     mockChain,
		BeaconDB:         beacondb,
		OperationService: &mockOperationService{},
		P2P:              simP2P,
		PowChainService:  mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	go ss.run()

	for ss.Querier.currentHeadSlot == 0 {
		simP2P.Send(simP2P.ctx, &pb.ChainHeadResponse{
			Slot:                      params.BeaconConfig().GenesisSlot + 12,
			Hash:                      []byte{'t', 'e', 's', 't'},
			FinalizedStateRootHash32S: stateRoot[:],
		}, "")
	}

	return ss, beacondb
}

func TestSyncing_AFullySyncedNode(t *testing.T) {
	numOfBlocks := 12
	ctx := context.Background()
	newP2P := &simulatedP2P{
		subsChannels: make(map[reflect.Type]*event.Feed),
		mutex:        new(sync.RWMutex),
		ctx:          ctx,
	}

	// Sets up a synced service which has its head at the current
	// numOfBlocks from genesis. The blocks are generated through
	// simulated backend.
	ss, syncedDB := setUpSyncedService(numOfBlocks, newP2P, t)
	defer ss.Stop()
	defer db.TeardownDB(syncedDB)

	bState, err := syncedDB.State(ctx)
	if err != nil {
		t.Fatalf("Could not retrieve state %v", err)
	}

	h, err := hashutil.HashProto(bState)
	if err != nil {
		t.Fatalf("unable to marshal the beacon state: %v", err)
	}

	// Sets up a sync service which has its current head at genesis.
	us, unSyncedDB := setUpUnSyncedService(newP2P, h, t)
	defer us.Stop()
	defer db.TeardownDB(unSyncedDB)

	// Sets up another sync service which has its current head at genesis.
	us2, unSyncedDB2 := setUpUnSyncedService(newP2P, h, t)
	defer us2.Stop()
	defer db.TeardownDB(unSyncedDB2)

	syncedChan := make(chan uint64)

	// Waits for the unsynced node to fire a message signifying it is
	// synced with its current slot number.
	sub := us.InitialSync.SyncedFeed().Subscribe(syncedChan)
	defer sub.Unsubscribe()

	syncedChan2 := make(chan uint64)

	sub2 := us2.InitialSync.SyncedFeed().Subscribe(syncedChan2)
	defer sub2.Unsubscribe()

	highestSlot := <-syncedChan

	highestSlot2 := <-syncedChan2

	if highestSlot != uint64(numOfBlocks)+params.BeaconConfig().GenesisSlot {
		t.Errorf("Sync services didn't sync to expectecd slot, expected %d but got %d",
			uint64(numOfBlocks)+params.BeaconConfig().GenesisSlot, highestSlot)
	}

	if highestSlot2 != uint64(numOfBlocks)+params.BeaconConfig().GenesisSlot {
		t.Errorf("Sync services didn't sync to expectecd slot, expected %d but got %d",
			uint64(numOfBlocks)+params.BeaconConfig().GenesisSlot, highestSlot2)
	}
}

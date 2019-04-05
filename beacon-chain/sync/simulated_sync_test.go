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
	ctx := context.Background()

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

	memBlocks := bd.InMemoryBlocks()
	if err := beacondb.SaveBlock(memBlocks[0]); err != nil {
		t.Fatalf("Could not save block %v", err)
	}
	if err := beacondb.SaveJustifiedBlock(memBlocks[0]); err != nil {
		t.Fatalf("Could not save block %v", err)
	}
	if err := beacondb.SaveFinalizedBlock(memBlocks[0]); err != nil {
		t.Fatalf("Could not save block %v", err)
	}

	state := bd.State()
	state.LatestBlock = memBlocks[0]
	state.LatestEth1Data = &pb.Eth1Data{
		BlockHash32: []byte{},
	}

	if err := beacondb.SaveState(ctx, state); err != nil {
		t.Fatalf("Could not save state %v", err)
	}
	if err := beacondb.SaveJustifiedState(state); err != nil {
		t.Fatalf("Could not save state %v", err)
	}
	if err := beacondb.SaveFinalizedState(state); err != nil {
		t.Fatalf("Could not save state %v", err)
	}

	if err := beacondb.UpdateChainHead(ctx, memBlocks[0], state); err != nil {
		t.Fatalf("Could not update chain head %v", err)
	}

	return bd, beacondb, privKeys
}

func setUpSyncedService(numOfBlocks int, simP2P *simulatedP2P, t *testing.T) (*Service, *db.BeaconDB, [32]byte) {
	bd, beacondb, _ := setupSimBackendAndDB(t)
	defer bd.Shutdown()
	defer db.TeardownDB(bd.DB())
	ctx := context.Background()

	mockPow := &genesisPowChain{
		feed: new(event.Feed),
	}

	mockChain := &mockChainService{
		bFeed: new(event.Feed),
		sFeed: new(event.Feed),
		cFeed: new(event.Feed),
		db:    bd.DB(),
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

	state, err := beacondb.State(ctx)
	if err != nil {
		t.Fatal(err)
	}
	inMemoryBlocks := bd.InMemoryBlocks()
	genesisBlock := inMemoryBlocks[0]
	stateRoot, err := hashutil.HashProto(state)
	if err != nil {
		t.Fatal(err)
	}
	parentRoot, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= numOfBlocks; i++ {
		block := &pb.BeaconBlock{
			Slot:             params.BeaconConfig().GenesisSlot + uint64(i),
			ParentRootHash32: parentRoot[:],
			StateRootHash32:  stateRoot[:],
		}
		state, err = mockChain.ApplyBlockStateTransition(ctx, block, state)
		if err != nil {
			t.Fatal(err)
		}
		stateRoot, err = hashutil.HashProto(state)
		if err != nil {
			t.Fatal(err)
		}
		parentRoot, err = hashutil.HashBeaconBlock(block)
		if err := mockChain.CleanupBlockOperations(ctx, block); err != nil {
			t.Fatal(err)
		}
		if err := beacondb.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		if err := beacondb.UpdateChainHead(ctx, block, state); err != nil {
			t.Fatal(err)
		}
	}
	return ss, beacondb, stateRoot
}

func setUpUnSyncedService(simP2P *simulatedP2P, stateRoot [32]byte, t *testing.T) (*Service, *db.BeaconDB) {
	bd, beacondb, _ := setupSimBackendAndDB(t)
	defer bd.Shutdown()
	defer db.TeardownDB(bd.DB())

	mockPow := &afterGenesisPowChain{
		feed: new(event.Feed),
	}

	mockChain := &mockChainService{
		bFeed: new(event.Feed),
		sFeed: new(event.Feed),
		cFeed: new(event.Feed),
		db:    bd.DB(),
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
			CanonicalSlot:            params.BeaconConfig().GenesisSlot + 12,
			CanonicalStateRootHash32: stateRoot[:],
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
	ss, syncedDB, stateRoot := setUpSyncedService(numOfBlocks, newP2P, t)
	defer ss.Stop()
	defer db.TeardownDB(syncedDB)

	// Sets up a sync service which has its current head at genesis.
	us, unSyncedDB := setUpUnSyncedService(newP2P, stateRoot, t)
	defer us.Stop()
	defer db.TeardownDB(unSyncedDB)

	us2, unSyncedDB2 := setUpUnSyncedService(newP2P, stateRoot, t)
	defer us2.Stop()
	defer db.TeardownDB(unSyncedDB2)

	finalized, err := syncedDB.FinalizedState()
	if err != nil {
		t.Fatal(err)
	}

	newP2P.Send(newP2P.ctx, &pb.BeaconStateResponse{
		FinalizedState: finalized,
	}, "")

	timeout := time.After(10 * time.Second)
	tick := time.Tick(200 * time.Millisecond)
loop:
	for {
		select {
		case <-timeout:
			t.Error("Could not sync in time")
			break loop
		case <-tick:
			_, slot1 := us.InitialSync.NodeIsSynced()
			_, slot2 := us2.InitialSync.NodeIsSynced()
			if slot1 == uint64(numOfBlocks)+params.BeaconConfig().GenesisSlot ||
				slot2 == uint64(numOfBlocks)+params.BeaconConfig().GenesisSlot {
				break loop
			}
		}
	}
}

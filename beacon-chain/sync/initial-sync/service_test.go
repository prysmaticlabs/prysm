package initialsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockP2P struct {
}

func (mp *mockP2P) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return new(event.Feed).Subscribe(channel)
}

func (mp *mockP2P) Broadcast(msg proto.Message) {}

func (mp *mockP2P) Send(msg proto.Message, peer p2p.Peer) {
}

type mockSyncService struct {
	hasStarted bool
	isSynced   bool
}

func (ms *mockSyncService) Start() {
	ms.hasStarted = true
}

func (ms *mockSyncService) IsSyncedWithNetwork() bool {
	return ms.isSynced
}

func (ms *mockSyncService) ResumeSync() {

}

type mockChainService struct{}

func (ms *mockChainService) IncomingBlockFeed() *event.Feed {
	return &event.Feed{}
}

func setUpGenesisStateAndBlock(beaconDB *db.BeaconDB, t *testing.T) {
	ctx := context.Background()
	genesisTime := time.Now()
	unixTime := uint64(genesisTime.Unix())
	if err := beaconDB.InitializeState(unixTime, []*pb.Deposit{}); err != nil {
		t.Fatalf("could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := beaconDB.State(ctx)
	if err != nil {
		t.Fatalf("could not attempt fetch beacon state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		log.Errorf("unable to marshal the beacon state: %v", err)
		return
	}
	genBlock := b.NewGenesisBlock(stateRoot[:])
	if err := beaconDB.SaveBlock(genBlock); err != nil {
		t.Fatalf("could not save genesis block to disk: %v", err)
	}
	if err := beaconDB.UpdateChainHead(genBlock, beaconState); err != nil {
		t.Fatalf("could not set chain head, %v", err)
	}
}

func TestSavingBlock_InSync(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	setUpGenesisStateAndBlock(db, t)

	cfg := &Config{
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		BeaconDB:     db,
		ChainService: &mockChainService{},
	}
	ss := NewInitialSyncService(context.Background(), cfg)
	ss.atGenesis = false

	exitRoutine := make(chan bool)
	delayChan := make(chan time.Time)

	defer func() {
		close(exitRoutine)
		close(delayChan)
	}()

	go func() {
		ss.run(delayChan)
		exitRoutine <- true
	}()

	genericHash := make([]byte, 32)
	genericHash[0] = 'a'

	beaconState := &pb.BeaconState{
		FinalizedEpoch: params.BeaconConfig().GenesisSlot + 1,
	}

	stateResponse := &pb.BeaconStateResponse{
		BeaconState: beaconState,
	}

	incorrectState := &pb.BeaconState{
		FinalizedEpoch: params.BeaconConfig().GenesisSlot,
		JustifiedEpoch: params.BeaconConfig().GenesisSlot + 1,
	}

	incorrectStateResponse := &pb.BeaconStateResponse{
		BeaconState: incorrectState,
	}

	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("unable to tree hash state: %v", err)
	}
	beaconStateRootHash32 := stateRoot

	getBlockResponseMsg := func(Slot uint64) p2p.Message {
		block := &pb.BeaconBlock{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{1, 2, 3},
				BlockHash32:       []byte{4, 5, 6},
			},
			ParentRootHash32: genericHash,
			Slot:             Slot,
			StateRootHash32:  beaconStateRootHash32[:],
		}

		blockResponse := &pb.BeaconBlockResponse{
			Block: block,
		}

		return p2p.Message{
			Peer: p2p.Peer{},
			Data: blockResponse,
		}
	}

	if err != nil {
		t.Fatalf("Unable to hash block %v", err)
	}

	msg1 := getBlockResponseMsg(params.BeaconConfig().GenesisSlot + 1)

	// saving genesis block
	ss.blockBuf <- msg1

	msg2 := p2p.Message{
		Peer: p2p.Peer{},
		Data: incorrectStateResponse,
	}

	ss.stateBuf <- msg2

	if ss.currentSlot == incorrectStateResponse.BeaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch {
		t.Fatalf("Beacon state updated incorrectly: %d", ss.currentSlot)
	}

	msg2.Data = stateResponse

	ss.stateBuf <- msg2

	msg1 = getBlockResponseMsg(params.BeaconConfig().GenesisSlot + 1)
	ss.blockBuf <- msg1
	if params.BeaconConfig().GenesisSlot+1 != ss.currentSlot {
		t.Fatalf("Slot saved when it was not supposed too: %v", stateResponse.BeaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch)
	}

	msg1 = getBlockResponseMsg(params.BeaconConfig().GenesisSlot + 2)
	ss.blockBuf <- msg1

	ss.cancel()
	<-exitRoutine

	br := msg1.Data.(*pb.BeaconBlockResponse)

	if br.Block.Slot != ss.currentSlot {
		t.Fatalf("Slot not updated despite receiving a valid block: %v", ss.currentSlot)
	}

	hook.Reset()
}

func TestDelayChan_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	setUpGenesisStateAndBlock(db, t)

	cfg := &Config{
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		BeaconDB:     db,
		ChainService: &mockChainService{},
	}
	ss := NewInitialSyncService(context.Background(), cfg)
	ss.atGenesis = false

	exitRoutine := make(chan bool)
	delayChan := make(chan time.Time)

	defer func() {
		close(exitRoutine)
		close(delayChan)
	}()

	go func() {
		ss.run(delayChan)
		exitRoutine <- true
	}()

	genericHash := make([]byte, 32)
	genericHash[0] = 'a'

	beaconState := &pb.BeaconState{
		FinalizedEpoch: params.BeaconConfig().GenesisSlot + 1,
	}

	stateResponse := &pb.BeaconStateResponse{
		BeaconState: beaconState,
	}

	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("unable to tree hash state: %v", err)
	}
	beaconStateRootHash32 := stateRoot

	block := &pb.BeaconBlock{
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{1, 2, 3},
			BlockHash32:       []byte{4, 5, 6},
		},
		ParentRootHash32: genericHash,
		Slot:             params.BeaconConfig().GenesisSlot + 1,
		StateRootHash32:  beaconStateRootHash32[:],
	}

	blockResponse := &pb.BeaconBlockResponse{
		Block: block,
	}

	msg1 := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse,
	}

	msg2 := p2p.Message{
		Peer: p2p.Peer{},
		Data: stateResponse,
	}

	ss.blockBuf <- msg1

	ss.stateBuf <- msg2

	blockResponse.Block.Slot = params.BeaconConfig().GenesisSlot + 1
	msg1.Data = blockResponse

	ss.blockBuf <- msg1

	delayChan <- time.Time{}

	ss.cancel()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "Exiting initial sync and starting normal sync")

	hook.Reset()
}

func TestRequestBlocksBySlot_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	setUpGenesisStateAndBlock(db, t)

	cfg := &Config{
		P2P:             &mockP2P{},
		SyncService:     &mockSyncService{},
		ChainService:    &mockChainService{},
		BeaconDB:        db,
		BlockBufferSize: 100,
	}
	ss := NewInitialSyncService(context.Background(), cfg)
	newState, err := state.GenesisBeaconState(nil, 0, nil)
	if err != nil {
		t.Fatalf("could not create new state %v", err)
	}

	err = ss.db.SaveState(newState)
	if err != nil {
		t.Fatalf("could not save beacon state %v", err)
	}

	ss.atGenesis = false

	exitRoutine := make(chan bool)
	delayChan := make(chan time.Time)

	defer func() {
		close(exitRoutine)
		close(delayChan)
	}()

	go func() {
		ss.run(delayChan)
		exitRoutine <- true
	}()
	genericHash := make([]byte, 32)
	genericHash[0] = 'a'

	getBlockResponseMsg := func(Slot uint64) (p2p.Message, [32]byte) {

		block := &pb.BeaconBlock{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{1, 2, 3},
				BlockHash32:       []byte{4, 5, 6},
			},
			ParentRootHash32: genericHash,
			Slot:             Slot,
			StateRootHash32:  nil,
		}

		blockResponse := &pb.BeaconBlockResponse{
			Block: block,
		}

		root, err := hashutil.HashBeaconBlock(block)
		if err != nil {
			t.Fatalf("unable to tree hash block %v", err)
		}

		return p2p.Message{
			Peer: p2p.Peer{},
			Data: blockResponse,
		}, root
	}

	// sending all blocks except for the genesis block
	startSlot := 1 + params.BeaconConfig().GenesisSlot
	for i := startSlot; i < startSlot+10; i++ {
		response, _ := getBlockResponseMsg(i)
		ss.blockBuf <- response
	}

	initialResponse, _ := getBlockResponseMsg(1 + params.BeaconConfig().GenesisSlot)

	//sending genesis block
	ss.blockBuf <- initialResponse

	_, hash := getBlockResponseMsg(9 + params.BeaconConfig().GenesisSlot)

	expString := fmt.Sprintf("Saved block with root %#x and slot %d for initial sync",
		hash, 9+params.BeaconConfig().GenesisSlot)

	// waiting for the current slot to come up to the
	// expected one.
	testutil.WaitForLog(t, hook, expString)

	delayChan <- time.Time{}

	ss.cancel()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "Exiting initial sync and starting normal sync")

	hook.Reset()
}

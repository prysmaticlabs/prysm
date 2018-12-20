package initialsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
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

func TestSetBlockForInitialSync(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	cfg := &Config{
		P2P:         &mockP2P{},
		SyncService: &mockSyncService{},
		BeaconDB:    db,
	}

	ss := NewInitialSyncService(context.Background(), cfg)

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

	block := &pb.BeaconBlock{
		CandidatePowReceiptRootHash32: []byte{1, 2, 3},
		ParentRootHash32:              genericHash,
		Slot:                          uint64(1),
		StateRootHash32:               genericHash,
	}

	blockResponse := &pb.BeaconBlockResponse{Block: block}
	msg1 := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse,
	}

	ss.blockBuf <- msg1

	ss.cancel()
	<-exitRoutine

	var hash [32]byte
	copy(hash[:], blockResponse.Block.StateRootHash32)

	if hash != ss.initialStateRootHash32 {
		t.Fatalf("Crystallized state hash not updated: %#x", blockResponse.Block.StateRootHash32)
	}

	hook.Reset()
}

func TestSavingBlocksInSync(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	cfg := &Config{
		P2P:         &mockP2P{},
		SyncService: &mockSyncService{},
		BeaconDB:    db,
	}
	ss := NewInitialSyncService(context.Background(), cfg)

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
		FinalizedSlot: 99,
	}

	stateResponse := &pb.BeaconStateResponse{
		BeaconState: beaconState,
	}

	incorrectState := &pb.BeaconState{
		FinalizedSlot: 9,
		JustifiedSlot: 20,
	}

	incorrectStateResponse := &pb.BeaconStateResponse{
		BeaconState: incorrectState,
	}

	beaconStateRootHash32, err := types.NewBeaconState(beaconState).Hash()
	if err != nil {
		t.Fatalf("unable to get hash of state: %v", err)
	}

	getBlockResponseMsg := func(Slot uint64) p2p.Message {
		block := &pb.BeaconBlock{
			CandidatePowReceiptRootHash32: []byte{1, 2, 3},
			ParentRootHash32:              genericHash,
			Slot:                          Slot,
			StateRootHash32:               beaconStateRootHash32[:],
		}

		blockResponse := &pb.BeaconBlockResponse{
			Block: block,
		}

		return p2p.Message{
			Peer: p2p.Peer{},
			Data: blockResponse,
		}
	}

	msg1 := getBlockResponseMsg(1)

	msg2 := p2p.Message{
		Peer: p2p.Peer{},
		Data: incorrectStateResponse,
	}

	ss.blockBuf <- msg1
	ss.stateBuf <- msg2

	if ss.currentSlot == incorrectStateResponse.BeaconState.FinalizedSlot {
		t.Fatalf("Crystallized state updated incorrectly: %d", ss.currentSlot)
	}

	msg2.Data = stateResponse

	ss.stateBuf <- msg2

	if beaconStateRootHash32 != ss.initialStateRootHash32 {
		br := msg1.Data.(*pb.BeaconBlockResponse)
		t.Fatalf("state hash not updated to: %#x instead it is %#x", br.Block.StateRootHash32,
			ss.initialStateRootHash32)
	}

	msg1 = getBlockResponseMsg(30)
	ss.blockBuf <- msg1

	if stateResponse.BeaconState.GetFinalizedSlot() != ss.currentSlot {
		t.Fatalf("Slot saved when it was not supposed too: %v", stateResponse.BeaconState.GetFinalizedSlot())
	}

	msg1 = getBlockResponseMsg(100)
	ss.blockBuf <- msg1

	ss.cancel()
	<-exitRoutine

	br := msg1.Data.(*pb.BeaconBlockResponse)
	if br.Block.GetSlot() != ss.currentSlot {
		t.Fatalf("Slot not updated despite receiving a valid block: %v", ss.currentSlot)
	}

	hook.Reset()
}

func TestDelayChan(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	cfg := &Config{
		P2P:         &mockP2P{},
		SyncService: &mockSyncService{},
		BeaconDB:    db,
	}
	ss := NewInitialSyncService(context.Background(), cfg)

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
		FinalizedSlot: 99,
	}

	stateResponse := &pb.BeaconStateResponse{
		BeaconState: beaconState,
	}

	beaconStateRootHash32, err := types.NewBeaconState(stateResponse.BeaconState).Hash()
	if err != nil {
		t.Fatalf("unable to get hash of state: %v", err)
	}

	block := &pb.BeaconBlock{
		CandidatePowReceiptRootHash32: []byte{1, 2, 3},
		ParentRootHash32:              genericHash,
		Slot:                          uint64(1),
		StateRootHash32:               beaconStateRootHash32[:],
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

	blockResponse.Block.Slot = 100
	msg1.Data = blockResponse

	ss.blockBuf <- msg1

	delayChan <- time.Time{}

	ss.cancel()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "Exiting initial sync and starting normal sync")

	hook.Reset()
}

func TestRequestBlocksBySlot(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	cfg := &Config{
		P2P:             &mockP2P{},
		SyncService:     &mockSyncService{},
		BeaconDB:        db,
		BlockBufferSize: 100,
	}
	ss := NewInitialSyncService(context.Background(), cfg)

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
			CandidatePowReceiptRootHash32: []byte{1, 2, 3},
			ParentRootHash32:              genericHash,
			Slot:                          Slot,
			StateRootHash32:               nil,
		}

		blockResponse := &pb.BeaconBlockResponse{
			Block: block,
		}

		hash, err := types.NewBlock(block).Hash()
		if err != nil {
			t.Fatalf("unable to hash block %v", err)
		}

		return p2p.Message{
			Peer: p2p.Peer{},
			Data: blockResponse,
		}, hash
	}

	// sending all blocks except for the initial block
	for i := uint64(2); i < 10; i++ {
		response, _ := getBlockResponseMsg(i)
		ss.blockBuf <- response
	}

	initialResponse, _ := getBlockResponseMsg(1)

	//sending initial block
	ss.blockBuf <- initialResponse

	_, hash := getBlockResponseMsg(9)

	expString := fmt.Sprintf("Saved block with hash %#x and slot %d for initial sync", hash, 9)

	// waiting for the current slot to come up to the
	// expected one.
	testutil.WaitForLog(t, hook, expString)

	delayChan <- time.Time{}

	ss.cancel()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "Exiting initial sync and starting normal sync")

	hook.Reset()
}

func TestRequestBatchedBlocks(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	cfg := &Config{
		P2P:             &mockP2P{},
		SyncService:     &mockSyncService{},
		BeaconDB:        db,
		BlockBufferSize: 100,
	}
	ss := NewInitialSyncService(context.Background(), cfg)

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

	getBlockResponse := func(Slot uint64) (*pb.BeaconBlockResponse, [32]byte) {

		block := &pb.BeaconBlock{
			CandidatePowReceiptRootHash32: []byte{1, 2, 3},
			ParentRootHash32:              genericHash,
			Slot:                          Slot,
			StateRootHash32:               nil,
		}

		blockResponse := &pb.BeaconBlockResponse{
			Block: block,
		}

		hash, err := types.NewBlock(block).Hash()
		if err != nil {
			t.Fatalf("unable to hash block %v", err)
		}

		return blockResponse, hash
	}

	for i := ss.currentSlot + 1; i <= 10; i++ {
		response, _ := getBlockResponse(i)
		ss.inMemoryBlocks[i] = response.Block
	}

	ss.requestBatchedBlocks(10)

	_, hash := getBlockResponse(10)
	expString := fmt.Sprintf("Saved block with hash %#x and slot %d for initial sync", hash, 10)

	// waiting for the current slot to come up to the
	// expected one.

	testutil.WaitForLog(t, hook, expString)

	delayChan <- time.Time{}

	ss.cancel()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "Exiting initial sync and starting normal sync")

	hook.Reset()
}

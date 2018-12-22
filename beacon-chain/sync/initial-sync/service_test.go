package initialsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
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

type mockChainService struct{}

func (ms *mockChainService) IncomingBlockFeed() *event.Feed {
	return &event.Feed{}
}

func TestSetBlockForInitialSync(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	cfg := &Config{
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		BeaconDB:     db,
		ChainService: &mockChainService{},
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
		Slot:                          uint64(0),
		StateRootHash32:               genericHash,
	}

	hash, err := b.Hash(block)
	if err != nil {
		t.Fatal(err)
	}

	ss.db.SaveBlock(block)
	ss.genesisHash = hash

	newblock := &pb.BeaconBlock{
		CandidatePowReceiptRootHash32: []byte{1, 2, 3},
		ParentRootHash32:              hash[:],
		Slot:                          uint64(1),
		StateRootHash32:               genericHash,
	}

	blockResponse := &pb.BeaconBlockResponse{Block: newblock}
	msg1 := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse,
	}

	ss.blockBuf <- msg1

	ss.cancel()
	<-exitRoutine

	var stateHash [32]byte
	copy(stateHash[:], blockResponse.Block.StateRootHash32)

	if stateHash != ss.initialStateRootHash32 {
		t.Fatalf("Beacon state hash not updated: %#x", blockResponse.Block.StateRootHash32)
	}

	hook.Reset()
}

func TestSavingBlocksInSync(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	cfg := &Config{
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		BeaconDB:     db,
		ChainService: &mockChainService{},
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

	getBlockResponseMsg := func(Slot uint64, parentHash [32]byte) p2p.Message {
		block := &pb.BeaconBlock{
			CandidatePowReceiptRootHash32: []byte{1, 2, 3},
			ParentRootHash32:              parentHash[:],
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

	msg0 := getBlockResponseMsg(0, [32]byte{})
	parentHash, err := b.Hash(msg0.Data.(*pb.BeaconBlockResponse).GetBlock())
	if err != nil {
		t.Fatalf("Unable to hash block %v", err)
	}

	msg1 := getBlockResponseMsg(1, parentHash)

	// saving genesis block
	ss.blockBuf <- msg1
	ss.blockBuf <- msg0

	msg2 := p2p.Message{
		Peer: p2p.Peer{},
		Data: incorrectStateResponse,
	}

	ss.stateBuf <- msg2

	if ss.currentSlot == incorrectStateResponse.BeaconState.FinalizedSlot {
		t.Fatalf("Beacon state updated incorrectly: %d", ss.currentSlot)
	}

	msg2.Data = stateResponse

	ss.stateBuf <- msg2

	if beaconStateRootHash32 != ss.initialStateRootHash32 {
		br := msg1.Data.(*pb.BeaconBlockResponse)
		t.Fatalf("state hash not updated to: %#x instead it is %#x", br.Block.StateRootHash32,
			ss.initialStateRootHash32)
	}

	msg1 = getBlockResponseMsg(30, [32]byte{})
	ss.blockBuf <- msg1

	if stateResponse.BeaconState.GetFinalizedSlot() != ss.currentSlot {
		t.Fatalf("Slot saved when it was not supposed too: %v", stateResponse.BeaconState.GetFinalizedSlot())
	}

	msg1 = getBlockResponseMsg(100, [32]byte{})
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
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		BeaconDB:     db,
		ChainService: &mockChainService{},
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

	genblock := &pb.BeaconBlock{
		CandidatePowReceiptRootHash32: []byte{1, 2, 3},
		ParentRootHash32:              genericHash,
		Slot:                          uint64(0),
		StateRootHash32:               genericHash,
	}

	hash, err := b.Hash(genblock)
	if err != nil {
		t.Fatal(err)
	}

	ss.db.SaveBlock(genblock)
	ss.genesisHash = hash

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
		ChainService:    &mockChainService{},
		BeaconDB:        db,
		BlockBufferSize: 100,
	}
	ss := NewInitialSyncService(context.Background(), cfg)
	newState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("could not create new state %v", err)
	}

	err = ss.db.SaveState(newState)
	if err != nil {
		t.Fatalf("Unable to save beacon state %v", err)
	}

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

	genblock := &pb.BeaconBlock{
		CandidatePowReceiptRootHash32: []byte{1, 2, 3},
		ParentRootHash32:              genericHash,
		Slot:                          uint64(0),
		StateRootHash32:               genericHash,
	}

	hash, err := b.Hash(genblock)
	if err != nil {
		t.Fatal(err)
	}

	ss.db.SaveBlock(genblock)
	ss.genesisHash = hash

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

		hash, err := b.Hash(block)
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

	_, hash = getBlockResponseMsg(9)

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

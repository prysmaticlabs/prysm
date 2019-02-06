package initialsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{1, 2, 3},
			BlockHash32:       []byte{4, 5, 6},
		},
		ParentRootHash32: genericHash,
		Slot:             uint64(1),
		StateRootHash32:  genericHash,
	}

	blockResponse := &pb.BeaconBlockResponse{Block: block}
	msg1 := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse,
	}

	ss.blockBuf <- msg1

	ss.cancel()
	<-exitRoutine

	stateHash := bytesutil.ToBytes32(blockResponse.Block.StateRootHash32)

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
		FinalizedEpoch: 1,
	}

	stateResponse := &pb.BeaconStateResponse{
		BeaconState: beaconState,
	}

	incorrectState := &pb.BeaconState{
		FinalizedEpoch: 0,
		JustifiedEpoch: 1,
	}

	incorrectStateResponse := &pb.BeaconStateResponse{
		BeaconState: incorrectState,
	}

	enc, err := proto.Marshal(beaconState)
	if err != nil {
		t.Fatalf("unable to get marshal state: %v", err)
	}
	beaconStateRootHash32 := hashutil.Hash(enc)

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

	msg1 := getBlockResponseMsg(1)

	// saving genesis block
	ss.blockBuf <- msg1

	msg2 := p2p.Message{
		Peer: p2p.Peer{},
		Data: incorrectStateResponse,
	}

	ss.stateBuf <- msg2

	if ss.currentSlot == incorrectStateResponse.BeaconState.FinalizedEpoch*params.BeaconConfig().EpochLength {
		t.Fatalf("Beacon state updated incorrectly: %d", ss.currentSlot)
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

	if stateResponse.BeaconState.FinalizedEpoch*params.BeaconConfig().EpochLength != ss.currentSlot {
		t.Fatalf("Slot saved when it was not supposed too: %v", stateResponse.BeaconState.FinalizedEpoch*params.BeaconConfig().EpochLength)
	}

	msg1 = getBlockResponseMsg(65)
	ss.blockBuf <- msg1

	ss.cancel()
	<-exitRoutine

	br := msg1.Data.(*pb.BeaconBlockResponse)
	if br.Block.Slot != ss.currentSlot {
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

	beaconState := &pb.BeaconState{
		FinalizedEpoch: 1,
	}

	stateResponse := &pb.BeaconStateResponse{
		BeaconState: beaconState,
	}

	enc, err := proto.Marshal(beaconState)
	if err != nil {
		t.Fatalf("unable to get marshal state: %v", err)
	}
	beaconStateRootHash32 := hashutil.Hash(enc)

	block := &pb.BeaconBlock{
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{1, 2, 3},
			BlockHash32:       []byte{4, 5, 6},
		},
		ParentRootHash32: genericHash,
		Slot:             uint64(1),
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

	blockResponse.Block.Slot = 65
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
	newState, err := state.InitialBeaconState(nil, 0, nil)
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

		hash, err := hashutil.HashBeaconBlock(block)
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

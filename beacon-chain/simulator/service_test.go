package simulator

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockP2P struct {
	broadcastHash []byte
}

func (mp *mockP2P) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return new(event.Feed).Subscribe(channel)
}

func (mp *mockP2P) Broadcast(msg proto.Message) {
	mp.broadcastHash = msg.(*pb.BeaconBlockAnnounce).Hash
}

func (mp *mockP2P) Send(msg proto.Message, peer p2p.Peer) {}

type mockPOWChainService struct{}

func (mpow *mockPOWChainService) LatestBlockHash() common.Hash {
	return common.BytesToHash([]byte{})
}

func setupSimulator(t *testing.T, beaconDB *db.BeaconDB) (*Simulator, *mockP2P) {
	ctx := context.Background()
	p2pService := &mockP2P{}

	err := beaconDB.InitializeState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	cfg := &Config{
		BlockRequestBuf: 0,
		P2P:             p2pService,
		Web3Service:     &mockPOWChainService{},
		BeaconDB:        beaconDB,
		EnablePOWChain:  true,
	}

	return NewSimulator(ctx, cfg), p2pService
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	sim, _ := setupSimulator(t, db)

	sim.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	sim.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")

	// The context should have been canceled.
	if sim.ctx.Err() == nil {
		t.Error("context was not canceled")
	}
}

func TestBroadcastBlockHash(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	sim, p2pService := setupSimulator(t, db)

	slotChan := make(chan uint64)
	exitRoutine := make(chan bool)

	go func() {
		sim.run(slotChan)
		<-exitRoutine
	}()

	// trigger a new block
	slotChan <- 1

	// test an invalid block request
	sim.blockRequestChan <- p2p.Message{
		Data: &pb.BeaconBlockRequest{
			Hash: make([]byte, 32),
		},
	}

	// test a valid block request
	blockHash := p2pService.broadcastHash
	sim.blockRequestChan <- p2p.Message{
		Data: &pb.BeaconBlockRequest{
			Hash: blockHash,
		},
	}

	// trigger another block
	slotChan <- 2

	testutil.AssertLogsContain(t, hook, "Broadcast block hash and slot")
	testutil.AssertLogsContain(t, hook, "Requested block not found")
	testutil.AssertLogsContain(t, hook, "Responding to full block request")

	// reset logs
	hook.Reset()

	// ensure that another request for the same block can be made
	sim.blockRequestChan <- p2p.Message{
		Data: &pb.BeaconBlockRequest{
			Hash: blockHash,
		},
	}

	sim.cancel()
	exitRoutine <- true

	testutil.AssertLogsDoNotContain(t, hook, "Requested block not found")
	testutil.AssertLogsContain(t, hook, "Responding to full block request")

	hook.Reset()
}

func TestBlockRequestBySlot(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	sim, _ := setupSimulator(t, db)

	slotChan := make(chan uint64)
	exitRoutine := make(chan bool)

	go func() {
		sim.run(slotChan)
		<-exitRoutine
	}()

	// trigger a new block
	slotChan <- 1

	// test an invalid block request
	sim.blockBySlotChan <- p2p.Message{
		Data: &pb.BeaconBlockRequestBySlotNumber{
			SlotNumber: 2,
		},
	}

	testutil.AssertLogsContain(t, hook, "Broadcast block hash and slot")
	testutil.AssertLogsContain(t, hook, "Requested block not found")

	// reset logs
	hook.Reset()

	// test a valid block request
	sim.blockBySlotChan <- p2p.Message{
		Data: &pb.BeaconBlockRequestBySlotNumber{
			SlotNumber: 1,
		},
	}

	sim.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Responding to full block request")

	hook.Reset()
}

func TestStateRequest(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	sim, _ := setupSimulator(t, db)

	slotChan := make(chan uint64)
	exitRoutine := make(chan bool)

	go func() {
		sim.run(slotChan)
		<-exitRoutine
	}()

	beaconState, err := sim.beaconDB.GetState()
	if err != nil {
		t.Fatalf("could not retrieve beacon state %v", err)
	}

	enc, err := proto.Marshal(beaconState)
	if err != nil {
		t.Fatalf("could not marshal beacon state %v", err)
	}
	hash := hashutil.Hash(enc)

	sim.stateReqChan <- p2p.Message{
		Data: &pb.BeaconStateRequest{
			Hash: []byte{'t', 'e', 's', 't'},
		},
	}

	testutil.WaitForLog(t, hook, "Requested beacon state is of a different hash")

	hook.Reset()

	sim.stateReqChan <- p2p.Message{
		Data: &pb.BeaconStateRequest{
			Hash: hash[:],
		},
	}

	testutil.WaitForLog(t, hook, "Responding to full beacon state request")

	sim.cancel()
	exitRoutine <- true
}

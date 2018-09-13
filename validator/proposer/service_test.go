package proposer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	shardingp2p "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/internal"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockClient struct {
	ctrl *gomock.Controller
}

func (mc *mockClient) ProposerServiceClient() pb.ProposerServiceClient {
	return internal.NewMockProposerServiceClient(mc.ctrl)
}

type mockAssigner struct{}

func (m *mockAssigner) ProposerAssignmentFeed() *event.Feed {
	return new(event.Feed)
}

type mockP2P struct {
}

func (mp *mockP2P) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return new(event.Feed).Subscribe(channel)
}

func (mp *mockP2P) Broadcast(msg proto.Message) {}

func (mp *mockP2P) Send(msg proto.Message, peer p2p.Peer) {
}

func TestSaveBlockToMemory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg, &mockP2P{})
	p.SaveBlockToMemory(&pbp2p.BeaconBlock{SlotNumber: 5})
	if p.blockMapping[5] == nil {
		t.Error("Block does not exist when it is supposed to")
	}
}

func TestGetBlockFromMemory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg, &mockP2P{})
	p.SaveBlockToMemory(&pbp2p.BeaconBlock{SlotNumber: 5, ParentHash: []byte{'t', 'e', 's', 't'}})
	block, err := p.GetBlockFromMemory(5)
	if err != nil {
		t.Fatalf("Unable to get block %v", err)
	}

	if !bytes.Equal(block.ParentHash, []byte{'t', 'e', 's', 't'}) {
		t.Error("parent hash not the same as the one saved")
	}

	block, err = p.GetBlockFromMemory(1)
	if err == nil {
		t.Error("block is able to be retrieved despite not existing")
	}

	p.blockMapping[2] = &pbp2p.BeaconBlock{SlotNumber: 8}
	block, err = p.GetBlockFromMemory(2)
	if err == nil {
		t.Error("block is regarded as valid despite being invalid")
	}

}
func TestDoesAttestationExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg, &mockP2P{})

	records := []*pbp2p.AttestationRecord{
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{'a'},
		},
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{'b'},
		},
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{'c'},
		},
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{'d'},
		}}

	fakeAttestation := &pbp2p.AttestationRecord{
		AttesterBitfield: []byte{'e'},
	}

	realAttestation := &pbp2p.AttestationRecord{
		AttesterBitfield: []byte{'a'},
	}

	if p.DoesAttestationExist(fakeAttestation, records) {
		t.Fatal("invalid attestation exists")
	}

	if !p.DoesAttestationExist(realAttestation, records) {
		t.Fatal("valid attestation does not exists")
	}

}

func TestAggregateSignatures(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg, &mockP2P{})

	records := []*pbp2p.AttestationRecord{
		&pbp2p.AttestationRecord{
			AggregateSig: []uint64{1},
		},
		&pbp2p.AttestationRecord{
			AggregateSig: []uint64{4},
		},
		&pbp2p.AttestationRecord{
			AggregateSig: []uint64{10},
		},
		&pbp2p.AttestationRecord{
			AggregateSig: []uint64{9},
		}}

	aggregatedSig := []uint32{1, 4, 10, 9}
	falseSig := []uint32{}

	if !reflect.DeepEqual(aggregatedSig, p.AggregateAllSignatures(records)) {
		t.Fatal("signatures have not been aggregated correctly")
	}

	if reflect.DeepEqual(falseSig, p.AggregateAllSignatures(records)) {
		t.Fatal("signatures that are meant to be unequal are showing up as equal")
	}

}

func TestGenerateBitmask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg, &mockP2P{})

	records := []*pbp2p.AttestationRecord{
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{2},
		},
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{4},
		},
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{8},
		},
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{16},
		}}
	falseMask := []byte{20}
	expectedMask := []byte{30}

	if bytes.Equal(p.GenerateBitmask(records), falseMask) {
		t.Fatalf("bitmasks that are equal are showing up as unequal %08b , %08b", p.GenerateBitmask(records), falseMask)
	}
	if !bytes.Equal(p.GenerateBitmask(records), expectedMask) {
		t.Fatalf("bitmasks that are the same are showing up as unequal %08b , %08b", p.GenerateBitmask(records), expectedMask)
	}
}
func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg, &mockP2P{})
	p.Start()
	p.Stop()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestProposerReceiveBeaconBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg, &mockP2P{})

	mockServiceClient := internal.NewMockProposerServiceClient(ctrl)

	doneChan := make(chan struct{})
	delayChan := make(chan time.Time)
	exitRoutine := make(chan bool)

	go func() {
		p.run(delayChan, doneChan, mockServiceClient)
		<-exitRoutine
	}()
	p.assignmentChan <- &pbp2p.BeaconBlock{SlotNumber: 5}
	doneChan <- struct{}{}
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Performing proposer responsibility")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Block proposed successfully with hash 0x%x", []byte("hi")))
	testutil.AssertLogsContain(t, hook, "Proposer context closed")
}

func TestProposerProcessAttestation(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg, &mockP2P{})

	mockServiceClient := internal.NewMockProposerServiceClient(ctrl)

	// Expect first call to go through correctly.
	mockServiceClient.EXPECT().ProposeBlock(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.ProposeResponse{
		BlockHash: []byte("hi"),
	}, nil)

	doneChan := make(chan struct{})
	delayChan := make(chan time.Time)
	exitRoutine := make(chan bool)

	go func() {
		p.run(delayChan, doneChan, mockServiceClient)
		<-exitRoutine
	}()
	p.assignmentChan <- &pbp2p.BeaconBlock{SlotNumber: 5}
	attestation := &shardingp2p.AttestationBroadcast{
		AttestationRecord: &pbp2p.AttestationRecord{Slot: 6},
	}
	msg := p2p.Message{Peer: p2p.Peer{}, Data: attestation}
	p.attestationBuf <- msg

	testutil.AssertLogsContain(t, hook, "Performing proposer responsibility")

	delayChan <- time.Time{}

	doneChan <- struct{}{}
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Block proposed successfully with hash 0x%x", []byte("hi")))
	testutil.AssertLogsContain(t, hook, "Proposer context closed")
}

func TestProposerServiceErrors(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg, &mockP2P{})

	mockServiceClient := internal.NewMockProposerServiceClient(ctrl)

	// Expect call to throw an error.
	mockServiceClient.EXPECT().ProposeBlock(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("bad block proposed"))

	doneChan := make(chan struct{})
	delayChan := make(chan time.Time)
	exitRoutine := make(chan bool)

	go func() {
		p.run(delayChan, doneChan, mockServiceClient)
		<-exitRoutine
	}()

	p.assignmentChan <- nil
	p.assignmentChan <- &pbp2p.BeaconBlock{SlotNumber: 5}

	attestation := &shardingp2p.AttestationBroadcast{
		AttestationRecord: &pbp2p.AttestationRecord{Slot: 5},
	}
	msg := p2p.Message{Peer: p2p.Peer{}, Data: attestation}
	p.attestationBuf <- msg

	attestation2 := &shardingp2p.AttestationBroadcast{
		AttestationRecord: &pbp2p.AttestationRecord{Slot: 6},
	}
	msg2 := p2p.Message{Peer: p2p.Peer{}, Data: attestation2}
	p.attestationBuf <- msg2
	delayChan <- time.Time{}

	msg3 := p2p.Message{Peer: p2p.Peer{}, Data: &pbp2p.BeaconBlock{}}
	p.attestationBuf <- msg3

	testutil.AssertLogsContain(t, hook, "Performing proposer responsibility")
	doneChan <- struct{}{}
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Could not marshal latest beacon block")
	testutil.AssertLogsContain(t, hook, "Could not propose block: bad block proposed")
	testutil.AssertLogsContain(t, hook, "Unable to retrieve block from memory block does not exist")
	testutil.AssertLogsContain(t, hook, "Received malformed attestation p2p message")
	testutil.AssertLogsContain(t, hook, "Proposer context closed")
}

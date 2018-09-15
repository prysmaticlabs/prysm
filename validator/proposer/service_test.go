package proposer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
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

func TestDoesAttestationExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg)

	p.pendingAttestation = []*pbp2p.AttestationRecord{
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

	if p.DoesAttestationExist(fakeAttestation) {
		t.Fatal("invalid attestation exists")
	}

	if !p.DoesAttestationExist(realAttestation) {
		t.Fatal("valid attestation does not exists")
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
	p := NewProposer(context.Background(), cfg)
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
	p := NewProposer(context.Background(), cfg)

	mockServiceClient := internal.NewMockProposerServiceClient(ctrl)
	mockServiceClient.EXPECT().ProposeBlock(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.ProposeResponse{
		BlockHash: []byte("hi"),
	}, nil)

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)

	go func() {
		p.run(doneChan, mockServiceClient)
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
	p := NewProposer(context.Background(), cfg)

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)

	go func() {
		p.processAttestation(doneChan)
		<-exitRoutine
	}()
	p.pendingAttestation = []*pbp2p.AttestationRecord{
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{'a'},
		},
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{'b'},
		}}

	attestation := &pbp2p.AttestationRecord{AttesterBitfield: []byte{'c'}}
	p.attestationChan <- attestation

	doneChan <- struct{}{}
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Attestation stored in memory")
	testutil.AssertLogsContain(t, hook, "Proposer context closed")

	if !bytes.Equal(p.pendingAttestation[2].GetAttesterBitfield(), []byte{'c'}) {
		t.Errorf("attestation was unable to be saved %v", p.pendingAttestation[2].GetAttesterBitfield())
	}
}

func TestFullProposalOfBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
		Client:        &mockClient{ctrl},
	}
	p := NewProposer(context.Background(), cfg)
	mockServiceClient := internal.NewMockProposerServiceClient(ctrl)
	mockServiceClient.EXPECT().ProposeBlock(
		gomock.Any(),
		gomock.Any(),
	).Return(&pb.ProposeResponse{
		BlockHash: []byte("hi"),
	}, nil)

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)

	go p.run(doneChan, mockServiceClient)

	go func() {
		p.processAttestation(doneChan)
		<-exitRoutine
	}()

	p.pendingAttestation = []*pbp2p.AttestationRecord{
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{'a'},
		},
		&pbp2p.AttestationRecord{
			AttesterBitfield: []byte{'b'},
		}}

	attestation := &pbp2p.AttestationRecord{AttesterBitfield: []byte{'c'}}
	p.attestationChan <- attestation

	p.assignmentChan <- &pbp2p.BeaconBlock{SlotNumber: 5}

	doneChan <- struct{}{}
	doneChan <- struct{}{}
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Performing proposer responsibility")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Block proposed successfully with hash 0x%x", []byte("hi")))
	testutil.AssertLogsContain(t, hook, "Proposer context closed")
	testutil.AssertLogsContain(t, hook, "Attestation stored in memory")
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
	p := NewProposer(context.Background(), cfg)

	mockServiceClient := internal.NewMockProposerServiceClient(ctrl)

	// Expect call to throw an error.
	mockServiceClient.EXPECT().ProposeBlock(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("bad block proposed"))

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)

	go p.run(doneChan, mockServiceClient)

	go func() {
		p.processAttestation(doneChan)
		<-exitRoutine
	}()

	p.attestationChan <- &pbp2p.AttestationRecord{}
	p.assignmentChan <- nil
	p.assignmentChan <- &pbp2p.BeaconBlock{SlotNumber: 9}

	doneChan <- struct{}{}
	doneChan <- struct{}{}
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Performing proposer responsibility")
	testutil.AssertLogsContain(t, hook, "Could not marshal latest beacon block")
	testutil.AssertLogsContain(t, hook, "Received malformed attestation p2p message")
	testutil.AssertLogsContain(t, hook, "Proposer context closed")
	testutil.AssertLogsContain(t, hook, "Could not propose block: bad block proposed")
}

package attester

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	shardingp2p "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockAssigner struct{}

func (m *mockAssigner) AttesterAssignmentFeed() *event.Feed {
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

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
	}
	att := NewAttester(context.Background(), cfg, &mockP2P{})
	att.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	att.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestAttesterLoop(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
	}
	att := NewAttester(context.Background(), cfg, &mockP2P{})

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		att.run(doneChan)
		<-exitRoutine
	}()
	block := &shardingp2p.BlockBroadcast{
		BeaconBlock: &pbp2p.BeaconBlock{}}

	att.assignmentChan <- true
	att.blockBuf <- p2p.Message{Peer: p2p.Peer{}, Data: &shardingp2p.AttestationBroadcast{}}
	att.blockBuf <- p2p.Message{Peer: p2p.Peer{}, Data: block}

	testutil.AssertLogsContain(t, hook, "Performing attester responsibility")
	doneChan <- struct{}{}
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Attestation Broadcasted to network")
	testutil.AssertLogsContain(t, hook, "Received malformed attestation p2p message")
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}

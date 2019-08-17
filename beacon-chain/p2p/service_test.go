package p2p

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockListener struct{}

func (m *mockListener) Self() *discv5.Node {
	panic("implement me")
}

func (m *mockListener) Close() {
	//no-op
}

func (m *mockListener) Lookup(discv5.NodeID) []*discv5.Node {
	panic("implement me")
}

func (m *mockListener) ReadRandomNodes([]*discv5.Node) int {
	panic("implement me")
}

func (m *mockListener) SetFallbackNodes([]*discv5.Node) error {
	panic("implement me")
}

func (m *mockListener) Resolve(discv5.NodeID) *discv5.Node {
	panic("implement me")
}

func (m *mockListener) RegisterTopic(discv5.Topic, <-chan struct{}) {
	panic("implement me")
}

func (m *mockListener) SearchTopic(discv5.Topic, <-chan time.Duration, chan<- *discv5.Node, chan<- bool) {
	panic("implement me")
}

func TestService_Stop_SetsStartedToFalse(t *testing.T) {
	s, _ := NewService(nil)
	s.started = true
	s.dv5Listener = &mockListener{}
	_ = s.Stop()

	if s.started != false {
		t.Error("Expected Service.started to be false, got true")
	}
}

func TestService_Start_OnlyStartsOnce(t *testing.T) {
	hook := logTest.NewGlobal()

	s, _ := NewService(&Config{})
	s.dv5Listener = &mockListener{}
	defer s.Stop()
	s.Start()
	if s.started != true {
		t.Error("Expected service to be started")
	}
	s.Start()
	testutil.AssertLogsContain(t, hook, "Attempted to start p2p service when it was already started")
}

func TestService_Status_NotRunning(t *testing.T) {
	s := &Service{started: false}
	s.dv5Listener = &mockListener{}
	if s.Status().Error() != "not running" {
		t.Errorf("Status returned wrong error, got %v", s.Status())
	}
}

func TestListenForNewNodes(t *testing.T) {
	//hook := logTest.NewGlobal()

	s, _ := NewService(&Config{})
	s.dv5Listener = &mockListener{}
	defer s.Stop()
	s.Start()
}

package p2p

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_Stop_SetsStartedToFalse(t *testing.T) {
	s, _ := NewService(nil)
	s.started = true
	_ = s.Stop()
	if s.started != false {
		t.Error("Expected Service.started to be false, got true")
	}
}

func TestService_Start_OnlyStartsOnce(t *testing.T) {
	hook := logTest.NewGlobal()

	s, _ := NewService(nil)
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
	if s.Status().Error() != "not running" {
		t.Errorf("Status returned wrong error, got %v", s.Status())
	}
}

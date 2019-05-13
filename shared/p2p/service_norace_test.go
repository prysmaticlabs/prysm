package p2p

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()

	s, err := NewServer(&ServerConfig{})
	if err != nil {
		t.Fatalf("Could not start a new server: %v", err)
	}

	s.Start()
	want := "Starting service"
	testutil.AssertLogsContain(t, hook, want)

	s.Stop()
	msg := hook.LastEntry().Message
	want = "Stopping service"
	if msg != want {
		t.Errorf("incorrect log. wanted: %s. got: %v", want, msg)
	}

	// The context should have been canceled.
	if s.ctx.Err() == nil {
		t.Error("Context was not cancelled")
	}
}

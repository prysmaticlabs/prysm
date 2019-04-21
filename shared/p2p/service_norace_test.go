package p2p

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{})
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()

	s, err := NewServer(&ServerConfig{})
	if err != nil {
		t.Fatalf("Could not start a new server: %v", err)
	}

	s.Start()
	msg := hook.Entries[0].Message
	want := "Starting service"
	if msg != want {
		t.Errorf("incorrect log. wanted: %s. got: %v", want, msg)
	}

	s.Stop()
	msg = hook.LastEntry().Message
	want = "Stopping service"
	if msg != want {
		t.Errorf("incorrect log. wanted: %s. got: %v", want, msg)
	}

	// The context should have been cancelled.
	if s.ctx.Err() == nil {
		t.Error("Context was not cancelled")
	}
}

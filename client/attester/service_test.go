package attester

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockBeaconService struct {
	proposerChan chan bool
	attesterChan chan bool
}

func (m *mockBeaconService) AttesterAssignment() <-chan bool {
	return m.attesterChan
}

func (m *mockBeaconService) ProposerAssignment() <-chan bool {
	return m.proposerChan
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := &mockBeaconService{
		attesterChan: make(chan bool),
	}
	at := NewAttester(context.Background(), b)
	at.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	at.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestAttesterLoop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := &mockBeaconService{
		attesterChan: make(chan bool),
	}
	at := NewAttester(context.Background(), b)

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		at.run(doneChan)
		<-exitRoutine
	}()
	b.attesterChan <- true
	testutil.AssertLogsContain(t, hook, "Performing attestation responsibility")
	doneChan <- struct{}{}
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Attester context closed")
}

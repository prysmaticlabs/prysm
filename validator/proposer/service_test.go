package proposer

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
		proposerChan: make(chan bool),
	}
	p := NewProposer(context.Background(), b)
	p.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	p.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestProposerLoop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	b := &mockBeaconService{
		proposerChan: make(chan bool),
	}
	p := NewProposer(context.Background(), b)

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		p.run(doneChan)
		<-exitRoutine
	}()
	b.proposerChan <- true
	testutil.AssertLogsContain(t, hook, "Performing proposer responsibility")
	doneChan <- struct{}{}
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Proposer context closed")
}

package attester

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/mock/gomock"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
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

func (mc *mockClient) AttesterServiceClient() pb.AttesterServiceClient {
	return internal.NewMockAttesterServiceClient(mc.ctrl)
}

type mockAssigner struct{}

func (m *mockAssigner) AttesterAssignmentFeed() *event.Feed {
	return new(event.Feed)
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := &Config{
		AssignmentBuf: 0,
		Assigner:      &mockAssigner{},
	}
	att := NewAttester(context.Background(), cfg)
	att.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	att.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

//func TestAttesterLoop(t *testing.T) {
//	hook := logTest.NewGlobal()
//	cfg := &Config{
//		AssignmentBuf: 0,
//		Assigner:      &mockAssigner{},
//	}
//	att := NewAttester(context.Background(), cfg)
//
//	doneChan := make(chan struct{})
//	exitRoutine := make(chan bool)
//	go func() {
//		att.run(doneChan)
//		<-exitRoutine
//	}()
//	att.assignmentChan <- true
//	testutil.AssertLogsContain(t, hook, "Performing attester responsibility")
//	doneChan <- struct{}{}
//	exitRoutine <- true
//	testutil.AssertLogsContain(t, hook, "Attester context closed")
//}

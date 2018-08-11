package proposer

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockBeaconService struct{}

func (m *mockBeaconService) AttesterAssignment() <-chan bool {
	return make(chan bool, 0)
}

func (m *mockBeaconService) ProposerAssignment() <-chan bool {
	return make(chan bool, 0)
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	prop := NewProposer(context.Background(), &mockBeaconService{})
	prop.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	prop.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

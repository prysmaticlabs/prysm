package proposer

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/client/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockClient struct {
	ctrl *gomock.Controller
}

func (fc *mockClient) BeaconServiceClient() pb.BeaconServiceClient {
	return internal.NewMockBeaconServiceClient(fc.ctrl)
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	prop := NewProposer(context.Background(), &mockClient{ctrl})
	prop.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	prop.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

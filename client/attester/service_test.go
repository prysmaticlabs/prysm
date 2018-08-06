package attester

import (
	"context"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type faultyClient struct{}

func (fc *faultyClient) BeaconServiceClient() pb.BeaconServiceClient {
	return NewMockBeaconServiceClient(nil)
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	client := &faultyClient{}
	at := NewAttester(context.Background(), client)
	at.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	at.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestFetchCrystallizedState(t *testing.T) {
	hook := logTest.NewGlobal()
	// Testing using a faulty client.
	client := &faultyClient{}
	at := NewAttester(context.Background(), client)

	at.fetchBeaconBlocks(client.BeaconServiceClient())

	testutil.AssertLogsContain(t, hook, "Could not setup beacon chain block streaming client")
}

func TestFetchBeaconHashHeight(t *testing.T) {
	hook := logTest.NewGlobal()
	// Testing using a faulty client.
	client := &faultyClient{}
	at := NewAttester(context.Background(), client)

	at.fetchBeaconBlocks(client.BeaconServiceClient())

	testutil.AssertLogsContain(t, hook, "Could not setup beacon chain block streaming client")
}

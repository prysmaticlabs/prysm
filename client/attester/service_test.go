package attester

import (
	"testing"

	"github.com/prysmaticlabs/prysm/client/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

type faultyClient struct{}

func (fc *faultyClient) BeaconServiceClient() pb.BeaconServiceClient {
	return internal.NewMockBeaconServiceClient(nil)
}

func TestLifecycle(t *testing.T) {
	// hook := logTest.NewGlobal()
	// at := NewAttester(context.Background(), &faultyClient{})
	// at.Start()
	// testutil.AssertLogsContain(t, hook, "Starting service")
	// at.Stop()
	// testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestFetchCrystallizedState(t *testing.T) {
	// hook := logTest.NewGlobal()
	// // Testing using a faulty client.
	// at := NewAttester(context.Background(), &faultyClient{})

	// ctrl := gomock.NewController(t)
	// defer ctrl.Finish()
	// mockService := internal.NewMockBeaconServiceClient(ctrl)

	// at.fetchBeaconBlocks(mockService)

	// testutil.AssertLogsContain(t, hook, "Could not setup beacon chain block streaming client")
}

func TestFetchBeaconHashHeight(t *testing.T) {
	// hook := logTest.NewGlobal()
	// // Testing using a faulty client.
	// at := NewAttester(context.Background(), &faultyClient{})

	// ctrl := gomock.NewController(t)
	// defer ctrl.Finish()
	// mockService := internal.NewMockBeaconServiceClient(ctrl)

	// at.fetchCrystallizedState(mockService)

	// testutil.AssertLogsContain(t, hook, "Could not setup beacon chain block streaming client")
}

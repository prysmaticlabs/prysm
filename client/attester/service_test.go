package attester

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/client/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
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
	hook := logTest.NewGlobal()
	// Testing using a faulty client.
	at := NewAttester(context.Background(), &faultyClient{})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock for the stream returned by LatestBeaconBlock.
	stream := internal.NewMockBeaconService_LatestBeaconBlockClient(ctrl)

	// Set expectation on receiving.
	stream.EXPECT().Recv().Return(&pbp2p.BeaconBlock{SlotNumber: 10}, nil)

	mockServiceClient := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestBeaconBlock(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	at.fetchBeaconBlocks(mockServiceClient)

	testutil.AssertLogsContain(t, hook, "Could not setup beacon chain block streaming client")
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

package attester

import (
	"context"
	"io"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/client/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type faultyClient struct {
	ctrl *gomock.Controller
}

func (fc *faultyClient) BeaconServiceClient() pb.BeaconServiceClient {
	return internal.NewMockBeaconServiceClient(fc.ctrl)
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	at := NewAttester(context.Background(), &faultyClient{ctrl})
	at.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestFetchBeaconBlocks(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	at := NewAttester(context.Background(), &faultyClient{ctrl})

	// Create mock for the stream returned by LatestBeaconBlock.
	stream := internal.NewMockBeaconService_LatestBeaconBlockClient(ctrl)

	// Set expectation on receiving.
	stream.EXPECT().Recv().Return(&pbp2p.BeaconBlock{SlotNumber: 10}, nil)
	stream.EXPECT().Recv().Return(&pbp2p.BeaconBlock{}, io.EOF)

	mockServiceClient := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestBeaconBlock(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	at.fetchBeaconBlocks(mockServiceClient)

	testutil.AssertLogsContain(t, hook, "Latest beacon block slot number")
}

func TestFetchCrystallizedState(t *testing.T) {
	// hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	at := NewAttester(context.Background(), &faultyClient{ctrl})

	// Create mock for the stream returned by LatestCrystallizedState.
	stream := internal.NewMockBeaconService_LatestCrystallizedStateClient(ctrl)

	// Set expectation on receiving.
	stream.EXPECT().Recv().Return(&pbp2p.CrystallizedState{}, nil)
	stream.EXPECT().Recv().Return(&pbp2p.CrystallizedState{}, io.EOF)

	mockServiceClient := internal.NewMockBeaconServiceClient(ctrl)
	mockServiceClient.EXPECT().LatestCrystallizedState(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	at.fetchCrystallizedState(mockServiceClient)

	// testutil.AssertLogsContain(t, hook, "Latest beacon block slot number")
}

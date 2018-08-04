package attester

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/grpc"
)

type faultyClient struct{}

func (fc *faultyClient) BeaconServiceClient() pb.BeaconServiceClient {
	return &faultyRPC{}
}

type faultyRPC struct{}

func (fr *faultyRPC) LatestBeaconBlock(ctx context.Context, emp *empty.Empty, opts ...grpc.CallOption) (pb.BeaconService_LatestBeaconBlockClient, error) {
	return nil, errors.New("error setting up")
}

func (fr *faultyRPC) LatestCrystallizedState(ctx context.Context, emp *empty.Empty, opts ...grpc.CallOption) (pb.BeaconService_LatestCrystallizedStateClient, error) {
	return nil, errors.New("error setting up")
}

func (fr *faultyRPC) ShuffleValidators(ctx context.Context, req *pb.ShuffleRequest, opts ...grpc.CallOption) (*pb.ShuffleResponse, error) {
	return nil, errors.New("error setting up")
}

func TestLifecycle(t *testing.T) {

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

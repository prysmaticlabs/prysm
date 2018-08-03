package attester

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"google.golang.org/grpc"
)

type mockBeaconServiceClient struct{}

func (mb *mockBeaconServiceClient) LatestBeaconBlock(ctx context.Context, emp *empty.Empty, opts ...grpc.CallOption) (pb.BeaconService_LatestBeaconBlockClient, error) {
	return nil, errors.New("error setting up")
}

func TestLifecycle(t *testing.T) {

}

func TestFetchCrystallizedState(t *testing.T) {

}

func TestFetchBeaconHashHeight(t *testing.T) {

}

package rpc

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	"google.golang.org/protobuf/types/known/timestamppb"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/client"
)

type mockSyncChecker struct {
	syncing bool
}

func (m *mockSyncChecker) Syncing(_ context.Context) (bool, error) {
	return m.syncing, nil
}

type mockGenesisFetcher struct{}

func (m *mockGenesisFetcher) GenesisInfo(_ context.Context) (*ethpb.Genesis, error) {
	genesis := timestamppb.New(time.Unix(0, 0))
	return &ethpb.Genesis{
		GenesisTime: genesis,
	}, nil
}

type mockBeaconInfoFetcher struct {
	endpoint string
}

func (m *mockBeaconInfoFetcher) BeaconLogsEndpoint(_ context.Context) (string, error) {
	return m.endpoint, nil
}

func TestServer_GetBeaconNodeConnection(t *testing.T) {
	ctx := context.Background()
	endpoint := "localhost:90210"
	vs, err := client.NewValidatorService(ctx, &client.Config{})
	require.NoError(t, err)
	s := &Server{
		walletInitialized:   true,
		validatorService:    vs,
		syncChecker:         &mockSyncChecker{syncing: false},
		genesisFetcher:      &mockGenesisFetcher{},
		nodeGatewayEndpoint: endpoint,
	}
	got, err := s.GetBeaconNodeConnection(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	want := &pb.NodeConnectionResponse{
		BeaconNodeEndpoint: endpoint,
		Connected:          false,
		Syncing:            false,
		GenesisTime:        uint64(time.Unix(0, 0).Unix()),
	}
	require.DeepEqual(t, want, got)
}

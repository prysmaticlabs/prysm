package rpc

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/client"
)

type mockSyncChecker struct {
	syncing bool
}

func (m *mockSyncChecker) Syncing(ctx context.Context) (bool, error) {
	return m.syncing, nil
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
		nodeGatewayEndpoint: endpoint,
	}
	got, err := s.GetBeaconNodeConnection(ctx, &ptypes.Empty{})
	require.NoError(t, err)

	want := &pb.NodeConnectionResponse{
		BeaconNodeEndpoint: endpoint,
		Connected:          false,
		Syncing:            false,
	}
	require.DeepEqual(t, want, got)
}

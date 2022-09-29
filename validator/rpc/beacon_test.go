package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	mock2 "github.com/prysmaticlabs/prysm/v3/testing/mock"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetBeaconStatus_NotConnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	nodeClient := mock2.NewMockNodeClient(ctrl)
	nodeClient.EXPECT().GetSyncStatus(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(nil /*response*/, errors.New("uh oh"))
	srv := &Server{
		beaconNodeClient: nodeClient,
	}
	ctx := context.Background()
	resp, err := srv.GetBeaconStatus(ctx, &empty.Empty{})
	require.NoError(t, err)
	want := &pb.BeaconStatusResponse{
		BeaconNodeEndpoint: "",
		Connected:          false,
		Syncing:            false,
	}
	assert.DeepEqual(t, want, resp)
}

func TestGetBeaconStatus_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	nodeClient := mock2.NewMockNodeClient(ctrl)
	beaconChainClient := mock2.NewMockBeaconChainClient(ctrl)
	nodeClient.EXPECT().GetSyncStatus(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.SyncStatus{Syncing: true}, nil)
	timeStamp := timestamppb.New(time.Unix(0, 0))
	nodeClient.EXPECT().GetGenesis(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.Genesis{
		GenesisTime:            timeStamp,
		DepositContractAddress: []byte("hello"),
	}, nil)
	beaconChainClient.EXPECT().GetChainHead(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.ChainHead{
		HeadEpoch: 1,
	}, nil)
	srv := &Server{
		beaconNodeClient:  nodeClient,
		beaconChainClient: beaconChainClient,
	}
	ctx := context.Background()
	resp, err := srv.GetBeaconStatus(ctx, &empty.Empty{})
	require.NoError(t, err)
	want := &pb.BeaconStatusResponse{
		BeaconNodeEndpoint:     "",
		Connected:              true,
		Syncing:                true,
		GenesisTime:            uint64(time.Unix(0, 0).Unix()),
		DepositContractAddress: []byte("hello"),
		ChainHead: &ethpb.ChainHead{
			HeadEpoch: 1,
		},
	}
	assert.DeepEqual(t, want, resp)
}

func TestGrpcHeaders(t *testing.T) {
	s := &Server{
		ctx:               context.Background(),
		clientGrpcHeaders: []string{"first=value1", "second=value2"},
	}
	err := s.registerBeaconClient()
	require.NoError(t, err)
	md, _ := metadata.FromOutgoingContext(s.ctx)
	require.Equal(t, 2, md.Len(), "MetadataV0 contains wrong number of values")
	assert.Equal(t, "value1", md.Get("first")[0])
	assert.Equal(t, "value2", md.Get("second")[0])
}

package beaconclient

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_ChainHead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		cfg: &Config{BeaconClient: client},
	}
	wanted := &ethpb.ChainHead{
		HeadSlot:      4,
		HeadEpoch:     0,
		HeadBlockRoot: make([]byte, 32),
	}
	client.EXPECT().GetChainHead(gomock.Any(), gomock.Any()).Return(wanted, nil)
	res, err := bs.ChainHead(context.Background())
	require.NoError(t, err)
	require.DeepEqual(t, wanted, res)
}

func TestService_GenesisValidatorsRoot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock.NewMockNodeClient(ctrl)
	bs := Service{
		cfg: &Config{NodeClient: client},
	}
	wanted := &ethpb.Genesis{
		GenesisValidatorsRoot: []byte("I am genesis"),
	}
	client.EXPECT().GetGenesis(gomock.Any(), gomock.Any()).Return(wanted, nil)
	res, err := bs.GenesisValidatorsRoot(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, wanted.GenesisValidatorsRoot, res, "Wanted %#x, received %#x", wanted.GenesisValidatorsRoot, res)
	// test next fetch uses memory and not the rpc call.
	res, err = bs.GenesisValidatorsRoot(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, wanted.GenesisValidatorsRoot, res, "Wanted %#x, received %#x", wanted.GenesisValidatorsRoot, res)
}

func TestService_QuerySyncStatus(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockNodeClient(ctrl)

	bs := Service{
		cfg: &Config{NodeClient: client},
	}
	syncStatusPollingInterval = time.Millisecond
	client.EXPECT().GetSyncStatus(gomock.Any(), gomock.Any()).Return(&ethpb.SyncStatus{
		Syncing: true,
	}, nil)
	client.EXPECT().GetSyncStatus(gomock.Any(), gomock.Any()).Return(&ethpb.SyncStatus{
		Syncing: false,
	}, nil)
	bs.querySyncStatus(context.Background())
	require.LogsContain(t, hook, "Waiting for beacon node to be fully synced...")
	require.LogsContain(t, hook, "Beacon node is fully synced")
}

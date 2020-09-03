package rpc

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2 "github.com/prysmaticlabs/prysm/validator/accounts/v2"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

func TestServer_CreateWallet(t *testing.T) {
	s := &Server{}
	resp, err := s.WalletConfig(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, resp, &pb.WalletResponse{})
}

func TestServer_WalletConfig_NoWalletFound(t *testing.T) {
	s := &Server{}
	resp, err := s.WalletConfig(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, resp, &pb.WalletResponse{})
}

func TestServer_WalletConfig(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	s := &Server{}
	// We attempt to create the wallet.
	_, err := v2.CreateWalletWithKeymanager(ctx, &v2.CreateWalletConfig{
		WalletCfg: &v2.WalletConfig{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: v2keymanager.Direct,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	resp, err := s.WalletConfig(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, resp, &pb.WalletResponse{
		WalletPath: localWalletDir,
	})
}

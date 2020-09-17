package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2 "github.com/prysmaticlabs/prysm/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func TestServer_CreateWallet_Direct(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	s := &Server{}
	req := &pb.CreateWalletRequest{
		WalletPath:        localWalletDir,
		Keymanager:        pb.CreateWalletRequest_DIRECT,
		WalletPassword:    strongPass,
		KeystoresPassword: strongPass,
	}
	_, err := s.CreateWallet(ctx, req)
	require.ErrorContains(t, "No keystores included for import", err)

	req.KeystoresImported = []string{"badjson"}
	_, err = s.CreateWallet(ctx, req)
	require.ErrorContains(t, "Not a valid EIP-2335 keystore", err)

	encryptor := keystorev4.New()
	keystores := make([]string, 3)
	for i := 0; i < len(keystores); i++ {
		privKey := bls.RandKey()
		pubKey := fmt.Sprintf("%x", privKey.PublicKey().Marshal())
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), strongPass)
		require.NoError(t, err)
		item := &v2keymanager.Keystore{
			Crypto:  cryptoFields,
			ID:      id.String(),
			Version: encryptor.Version(),
			Pubkey:  pubKey,
			Name:    encryptor.Name(),
		}
		encodedFile, err := json.MarshalIndent(item, "", "\t")
		require.NoError(t, err)
		keystores[i] = string(encodedFile)
	}
	req.KeystoresImported = keystores
	_, err = s.CreateWallet(ctx, req)
	require.NoError(t, err)
}

func TestServer_CreateWallet_Derived(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	s := &Server{}
	req := &pb.CreateWalletRequest{
		WalletPath:     localWalletDir,
		Keymanager:     pb.CreateWalletRequest_DERIVED,
		WalletPassword: strongPass,
		NumAccounts:    0,
	}
	_, err := s.CreateWallet(ctx, req)
	require.ErrorContains(t, "Must create at least 1 validator account", err)

	req.NumAccounts = 2
	_, err = s.CreateWallet(ctx, req)
	require.ErrorContains(t, "Must include mnemonic", err)

	mnemonicResp, err := s.GenerateMnemonic(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	req.Mnemonic = mnemonicResp.Mnemonic

	_, err = s.CreateWallet(ctx, req)
	require.NoError(t, err)
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
		WalletCfg: &wallet.Config{
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

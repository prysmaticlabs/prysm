package accounts

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	mock2 "github.com/prysmaticlabs/prysm/v3/testing/mock"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestExitAccountsCli_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockValidatorClient := mock2.NewMockBeaconNodeValidatorClient(ctrl)
	mockNodeClient := mock2.NewMockNodeClient(ctrl)

	mockValidatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(&ethpb.ValidatorIndexResponse{Index: 1}, nil)

	// Any time in the past will suffice
	genesisTime := &timestamppb.Timestamp{
		Seconds: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}

	mockNodeClient.EXPECT().
		GetGenesis(gomock.Any(), gomock.Any()).
		Return(&ethpb.Genesis{GenesisTime: genesisTime}, nil)

	mockValidatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil)

	mockValidatorClient.EXPECT().
		ProposeExit(gomock.Any(), gomock.AssignableToTypeOf(&ethpb.SignedVoluntaryExit{})).
		Return(&ethpb.ProposeExitResponse{}, nil)

	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	// Write a directory where we will import keys from.
	keysDir := filepath.Join(t.TempDir(), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	// Create keystore file in the keys directory we can then import from in our wallet.
	keystore, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)

	// We initialize a wallet with a local keymanager.
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		// Wallet configuration flags.
		walletDir:           walletDir,
		keymanagerKind:      keymanager.Local,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
		// Flag required for ImportAccounts to work.
		keysDir: keysDir,
		// Flag required for ExitAccounts to work.
		voluntaryExitPublicKeys: keystore.Pubkey,
	})
	_, err := accounts.CreateWalletWithKeymanager(cliCtx.Context, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Local,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)
	require.NoError(t, accountsImport(cliCtx))

	_, km, err := walletWithKeymanager(cliCtx)
	require.NoError(t, err)
	require.NotNil(t, km)

	validatingPublicKeys, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)
	require.NotNil(t, validatingPublicKeys)

	// Prepare user input for final confirmation step
	var stdin bytes.Buffer
	stdin.Write([]byte(accounts.ExitPassphrase))
	rawPubKeys, formattedPubKeys, err := accounts.FilterExitAccountsFromUserInput(
		cliCtx, &stdin, validatingPublicKeys,
	)
	require.NoError(t, err)
	require.NotNil(t, rawPubKeys)
	require.NotNil(t, formattedPubKeys)

	cfg := accounts.PerformExitCfg{
		ValidatorClient:  mockValidatorClient,
		NodeClient:       mockNodeClient,
		Keymanager:       km,
		RawPubKeys:       rawPubKeys,
		FormattedPubKeys: formattedPubKeys,
	}
	rawExitedKeys, formattedExitedKeys, err := accounts.PerformVoluntaryExit(cliCtx.Context, cfg)
	require.NoError(t, err)
	require.Equal(t, 1, len(rawExitedKeys))
	assert.DeepEqual(t, rawPubKeys[0], rawExitedKeys[0])
	require.Equal(t, 1, len(formattedExitedKeys))
	assert.Equal(t, "0x"+keystore.Pubkey[:12], formattedExitedKeys[0])
}

func TestExitAccountsCli_OK_AllPublicKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockValidatorClient := mock2.NewMockBeaconNodeValidatorClient(ctrl)
	mockNodeClient := mock2.NewMockNodeClient(ctrl)

	mockValidatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(&ethpb.ValidatorIndexResponse{Index: 0}, nil)

	mockValidatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(&ethpb.ValidatorIndexResponse{Index: 1}, nil)

	// Any time in the past will suffice
	genesisTime := &timestamppb.Timestamp{
		Seconds: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}

	mockNodeClient.EXPECT().
		GetGenesis(gomock.Any(), gomock.Any()).
		Times(2).
		Return(&ethpb.Genesis{GenesisTime: genesisTime}, nil)

	mockValidatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Times(2).
		Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil)

	mockValidatorClient.EXPECT().
		ProposeExit(gomock.Any(), gomock.AssignableToTypeOf(&ethpb.SignedVoluntaryExit{})).
		Times(2).
		Return(&ethpb.ProposeExitResponse{}, nil)

	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	// Write a directory where we will import keys from.
	keysDir := filepath.Join(t.TempDir(), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	// Create keystore file in the keys directory we can then import from in our wallet.
	keystore1, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	keystore2, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)

	// We initialize a wallet with a local keymanager.
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		// Wallet configuration flags.
		walletDir:           walletDir,
		keymanagerKind:      keymanager.Local,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
		// Flag required for ImportAccounts to work.
		keysDir: keysDir,
		// Exit all public keys.
		exitAll: true,
	})
	_, err := accounts.CreateWalletWithKeymanager(cliCtx.Context, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Local,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)
	require.NoError(t, accountsImport(cliCtx))

	_, km, err := walletWithKeymanager(cliCtx)
	require.NoError(t, err)
	require.NotNil(t, km)

	validatingPublicKeys, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)
	require.NotNil(t, validatingPublicKeys)

	// Prepare user input for final confirmation step
	var stdin bytes.Buffer
	stdin.Write([]byte(accounts.ExitPassphrase))
	rawPubKeys, formattedPubKeys, err := accounts.FilterExitAccountsFromUserInput(
		cliCtx, &stdin, validatingPublicKeys,
	)
	require.NoError(t, err)
	require.NotNil(t, rawPubKeys)
	require.NotNil(t, formattedPubKeys)

	cfg := accounts.PerformExitCfg{
		ValidatorClient:  mockValidatorClient,
		NodeClient:       mockNodeClient,
		Keymanager:       km,
		RawPubKeys:       rawPubKeys,
		FormattedPubKeys: formattedPubKeys,
	}
	rawExitedKeys, formattedExitedKeys, err := accounts.PerformVoluntaryExit(cliCtx.Context, cfg)
	require.NoError(t, err)
	require.Equal(t, 2, len(rawExitedKeys))
	assert.DeepEqual(t, rawPubKeys, rawExitedKeys)
	require.Equal(t, 2, len(formattedExitedKeys))
	wantedFormatted := []string{
		"0x" + keystore1.Pubkey[:12],
		"0x" + keystore2.Pubkey[:12],
	}
	sort.Strings(wantedFormatted)
	sort.Strings(formattedExitedKeys)
	require.DeepEqual(t, wantedFormatted, formattedExitedKeys)
}

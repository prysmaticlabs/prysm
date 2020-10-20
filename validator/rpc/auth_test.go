package rpc

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

func setupWalletDir(t testing.TB) string {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	walletDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "wallet")
	require.NoError(t, os.MkdirAll(walletDir, os.ModePerm))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(walletDir), "Failed to remove directory")
	})
	return walletDir
}

func TestServer_SignupAndLogin_RoundTrip(t *testing.T) {
	valDB := dbtest.SetupDB(t, [][48]byte{})
	ctx := context.Background()

	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	strongPass := "29384283xasjasd32%%&*@*#*"

	ss := &Server{
		valDB:                 valDB,
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
	}
	weakPass := "password"
	_, err := ss.Signup(ctx, &pb.AuthRequest{
		Password: weakPass,
	})
	require.ErrorContains(t, "Could not validate RPC password input", err)

	// We assert we are able to signup with a strong password.
	_, err = ss.Signup(ctx, &pb.AuthRequest{
		Password: strongPass,
	})
	require.NoError(t, err)

	// Assert we stored the hashed password.
	passwordHashExists := fileutil.FileExists(filepath.Join(defaultWalletPath, HashedRPCPassword))
	assert.Equal(t, true, passwordHashExists)

	// We attempt to create the wallet.
	_, err = accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)

	// We assert we are able to login.
	_, err = ss.Login(ctx, &pb.AuthRequest{
		Password: strongPass,
	})
	require.NoError(t, err)
}

func TestServer_ChangePassword_Preconditions(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	strongPass := "29384283xasjasd32%%&*@*#*"
	ss := &Server{
		walletDir: defaultWalletPath,
	}
	require.NoError(t, ss.SaveHashedPassword(strongPass))
	_, err := ss.ChangePassword(ctx, &pb.ChangePasswordRequest{
		CurrentPassword: strongPass,
		Password:        "",
	})
	assert.ErrorContains(t, "Could not validate password input", err)
	_, err = ss.ChangePassword(ctx, &pb.ChangePasswordRequest{
		CurrentPassword:      strongPass,
		Password:             "abc",
		PasswordConfirmation: "def",
	})
	assert.ErrorContains(t, "does not match", err)
}

func TestServer_ChangePassword_OK(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ss := &Server{
		walletDir: defaultWalletPath,
	}
	password := "Passw0rdz%%%%pass"
	newPassword := "NewPassw0rdz%%%%pass"
	ctx := context.Background()
	require.NoError(t, ss.SaveHashedPassword(password))
	_, err := ss.ChangePassword(ctx, &pb.ChangePasswordRequest{
		CurrentPassword:      password,
		Password:             newPassword,
		PasswordConfirmation: newPassword,
	})
	require.NoError(t, err)
	_, err = ss.Login(ctx, &pb.AuthRequest{
		Password: newPassword,
	})
	require.NoError(t, err)
}

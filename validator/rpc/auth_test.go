package rpc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

func setupWalletDir(t testing.TB) string {
	walletDir := filepath.Join(t.TempDir(), "wallet")
	require.NoError(t, os.MkdirAll(walletDir, os.ModePerm))
	return walletDir
}

func TestServer_SignupAndLogin_RoundTrip(t *testing.T) {
	valDB := dbtest.SetupDB(t, [][48]byte{})
	ctx := context.Background()

	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir

	ss := &Server{
		valDB:                 valDB,
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
	}
	weakPass := "password"
	_, err := ss.Signup(ctx, &pb.AuthRequest{
		Password:             weakPass,
		PasswordConfirmation: weakPass,
	})
	require.ErrorContains(t, "Could not validate RPC password input", err)

	// We assert we are able to signup with a strong password.
	_, err = ss.Signup(ctx, &pb.AuthRequest{
		Password:             strongPass,
		PasswordConfirmation: strongPass,
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

func TestServer_Logout(t *testing.T) {
	key, err := createRandomJWTKey()
	require.NoError(t, err)
	ss := &Server{
		jwtKey: key,
	}
	tokenString, _, err := ss.createTokenString()
	require.NoError(t, err)
	checkParsedKey := func(*jwt.Token) (interface{}, error) {
		return ss.jwtKey, nil
	}
	_, err = jwt.Parse(tokenString, checkParsedKey)
	assert.NoError(t, err)

	_, err = ss.Logout(context.Background(), &empty.Empty{})
	require.NoError(t, err)

	// Attempting to validate the same token string after logout should fail.
	_, err = jwt.Parse(tokenString, checkParsedKey)
	assert.ErrorContains(t, "signature is invalid", err)
}

func TestServer_ChangePassword_Preconditions(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
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

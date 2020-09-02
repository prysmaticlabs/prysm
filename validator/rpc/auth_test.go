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
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2 "github.com/prysmaticlabs/prysm/validator/accounts/v2"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

func setupWalletDir(t testing.TB) string {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	walletDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "wallet")
	require.NoError(t, os.RemoveAll(walletDir), "Failed to remove directory")
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(walletDir), "Failed to remove directory")
	})
	return walletDir
}

func TestServer_Signup_PasswordAlreadyExists(t *testing.T) {
	valDB := dbtest.SetupDB(t, [][48]byte{})
	ctx := context.Background()
	ss := &Server{
		valDB: valDB,
	}

	// Save a hash password pre-emptively to the database.
	hashedPassword := []byte("2093402934902839489238492")
	require.NoError(t, valDB.SaveHashedPasswordForAPI(ctx, hashedPassword))

	// Attempt to signup despite already having a hashed password in the DB
	// which should immediately fail.
	strongPass := "29384283xasjasd32%%&*@*#*"
	_, err := ss.Signup(ctx, &pb.AuthRequest{
		Password: strongPass,
	})
	require.ErrorContains(t, "Validator already has a password set, cannot signup", err)
}

func TestServer_SignupAndLogin_RoundTrip(t *testing.T) {
	valDB := dbtest.SetupDB(t, [][48]byte{})
	ctx := context.Background()

	walletDir := setupWalletDir(t)
	strongPass := "29384283xasjasd32%%&*@*#*"
	// We attempt to create the wallet.
	_, err := v2.CreateWalletWithKeymanager(ctx, &v2.CreateWalletConfig{
		WalletCfg: &v2.WalletConfig{
			WalletDir:      walletDir,
			KeymanagerKind: v2keymanager.Direct,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)

	ss := &Server{
		valDB:                 valDB,
		walletInitializedFeed: new(event.Feed),
	}
	weakPass := "password"
	_, err = ss.Signup(ctx, &pb.AuthRequest{
		Password: weakPass,
	})
	require.ErrorContains(t, "Could not validate password input", err)

	// We assert we are able to signup with a strong password.
	_, err = ss.Signup(ctx, &pb.AuthRequest{
		Password: strongPass,
	})
	require.NoError(t, err)

	// Assert we stored the hashed password.
	hashedPass, err := valDB.HashedPasswordForAPI(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, 0, len(hashedPass))

	// We assert we are able to login.
	_, err = ss.Login(ctx, &pb.AuthRequest{
		Password: strongPass,
	})
	require.NoError(t, err)
}

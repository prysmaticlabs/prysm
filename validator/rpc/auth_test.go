package rpc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
)

func setupWalletDir(t testing.TB) string {
	walletDir := filepath.Join(t.TempDir(), "wallet")
	require.NoError(t, os.MkdirAll(walletDir, os.ModePerm))
	return walletDir
}

//func TestServer_SignupAndLogin_RoundTrip(t *testing.T) {
//valDB := dbtest.SetupDB(t, [][48]byte{})
//ctx := context.Background()

//localWalletDir := setupWalletDir(t)
//defaultWalletPath = localWalletDir

//ss := &Server{
//valDB:                 valDB,
//walletInitializedFeed: new(event.Feed),
//walletDir:             defaultWalletPath,
//}
//weakPass := "password"
//_, err := ss.Signup(ctx, &pb.AuthRequest{
//Password:             weakPass,
//PasswordConfirmation: weakPass,
//})
//require.ErrorContains(t, "Could not validate RPC password input", err)

//// We assert we are able to signup with a strong password.
//_, err = ss.Signup(ctx, &pb.AuthRequest{
//Password:             strongPass,
//PasswordConfirmation: strongPass,
//})
//require.NoError(t, err)

//// Assert we stored the hashed password.
//passwordHashExists := file.FileExists(filepath.Join(defaultWalletPath, HashedRPCPassword))
//assert.Equal(t, true, passwordHashExists)

//// We attempt to create the wallet.
//_, err = accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
//WalletCfg: &wallet.Config{
//WalletDir:      defaultWalletPath,
//KeymanagerKind: keymanager.Derived,
//WalletPassword: strongPass,
//},
//SkipMnemonicConfirm: true,
//})
//require.NoError(t, err)

//// We assert we are able to login.
//_, err = ss.Login(ctx, &pb.AuthRequest{
//Password: strongPass,
//})
//require.NoError(t, err)
//}

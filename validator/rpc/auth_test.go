package rpc

import (
	"bytes"
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

func Test_initializeAuthToken(t *testing.T) {
	// Initializing for the first time, there is no auth token file in
	// the wallet directory, so we generate a jwt token and secret from scratch.
	srv := &Server{}
	walletDir := setupWalletDir(t)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(walletDir))
	})
	token, expr, err := srv.initializeAuthToken(walletDir)
	require.NoError(t, err)
	require.Equal(t, true, len(srv.jwtKey) > 0)

	// Initializing second time, we generate something from the initial file.
	srv2 := &Server{}
	token2, expr2, err := srv2.initializeAuthToken(walletDir)
	require.NoError(t, err)
	require.Equal(t, true, bytes.Equal(srv.jwtKey, srv2.jwtKey))
	require.Equal(t, token, token2)
	require.Equal(t, expr, expr2)

	// Deleting the auth token and re-initializing means we create a jwt token
	// and secret from scratch again.
	require.NoError(t, os.RemoveAll(walletDir))
	srv3 := &Server{}
	walletDir = setupWalletDir(t)
	token3, expr3, err := srv3.initializeAuthToken(walletDir)
	require.NoError(t, err)
	require.Equal(t, true, len(srv.jwtKey) > 0)
	require.NotEqual(t, token, token3)
	require.NotEqual(t, token, expr3)
}

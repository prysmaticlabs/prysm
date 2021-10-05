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

package v2

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
)

func TestBackupAccounts_Noninteractive(t *testing.T) {
	walletDir, _, passwordFilePath := setupWalletAndPasswordsDir(t)
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random file path")
	// Write a directory where we will import keys from.
	keysDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "keysDir")
	require.NoError(t, os.MkdirAll(keysDir, os.ModePerm))

	// Write a directory where we will backup accounts to.
	backupDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath), "backupDir")
	require.NoError(t, os.MkdirAll(backupDir, os.ModePerm))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(keysDir), "Failed to remove directory")
		require.NoError(t, os.RemoveAll(backupDir), "Failed to remove directory")
	})

	// Create 2 keystore files in the keys directory we can then
	// import from in our wallet.
	k1, _ := createKeystore(t, keysDir)
	time.Sleep(time.Second)
	k2, _ := createKeystore(t, keysDir)
	generatedPubKeys := []string{k1.Pubkey, k2.Pubkey}
	backupForPublicKeys := strings.Join(generatedPubKeys, ",")

	// Write a password for the accounts we wish to backup to a file.
	backupsPasswordFile := filepath.Join(backupDir, "backuppass.txt")
	err = ioutil.WriteFile(
		backupsPasswordFile,
		[]byte("Passw0rdz4938%%"),
		params.BeaconIoConfig().ReadWritePermissions,
	)
	require.NoError(t, err)

	// We initialize a wallet with a direct keymanager.
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		// Wallet configuration flags.
		walletDir:           walletDir,
		keymanagerKind:      v2keymanager.Direct,
		walletPasswordFile:  passwordFilePath,
		accountPasswordFile: passwordFilePath,
		// Flags required for ImportAccounts to work.
		keysDir: keysDir,
		// Flags required for BackupAccounts to work.
		backupForPublicKeys: backupForPublicKeys,
		backupPasswordFile:  backupsPasswordFile,
		backupDir:           backupDir,
	})
	wallet, err := NewWallet(cliCtx, v2keymanager.Direct)
	require.NoError(t, err)
	require.NoError(t, wallet.SaveWallet())
	ctx := context.Background()
	encodedCfg, err := direct.MarshalConfigFile(ctx, direct.DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg))

	// We attempt to import accounts we wrote to the keys directory
	// into our newly created wallet.
	require.NoError(t, ImportAccounts(cliCtx))

	// Next, we attempt to backup the accounts.
	require.NoError(t, BackupAccounts(cliCtx))

	// We check a backup.zip file was created at the output path.
	zipFilePath := filepath.Join(backupDir, archiveFilename)
	assert.DeepEqual(t, true, fileutil.FileExists(zipFilePath))

	// We attempt to unzip the file and verify the keystores do match our accounts.
	f, err := os.Open(zipFilePath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()
	fi, err := f.Stat()
	require.NoError(t, err)
	r, err := zip.NewReader(f, fi.Size())
	require.NoError(t, err)

	// We check we have 2 keystore files in the unzipped results.
	require.DeepEqual(t, 2, len(r.File))
	unzippedPublicKeys := make([]string, 2)
	for i, unzipped := range r.File {
		ff, err := unzipped.Open()
		require.NoError(t, err)
		encodedBytes, err := ioutil.ReadAll(ff)
		require.NoError(t, err)
		keystoreFile := &v2keymanager.Keystore{}
		require.NoError(t, json.Unmarshal(encodedBytes, keystoreFile))
		require.NoError(t, ff.Close())
		unzippedPublicKeys[i] = keystoreFile.Pubkey
	}
	sort.Strings(unzippedPublicKeys)
	sort.Strings(generatedPubKeys)
	assert.DeepEqual(t, unzippedPublicKeys, generatedPubKeys)
}

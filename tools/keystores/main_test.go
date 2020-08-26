package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

type cliConfig struct {
	keystoresPath string
	password      string
	privateKey    string
	outputPath    string
}

func setupCliContext(
	tb testing.TB,
	conf *cliConfig,
) *cli.Context {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(keystoresFlag.Name, conf.keystoresPath, "")
	set.String(passwordFlag.Name, conf.password, "")
	set.String(privateKeyFlag.Name, conf.privateKey, "")
	set.String(outputPathFlag.Name, conf.outputPath, "")
	assert.NoError(tb, set.Set(keystoresFlag.Name, conf.keystoresPath))
	assert.NoError(tb, set.Set(passwordFlag.Name, conf.password))
	assert.NoError(tb, set.Set(privateKeyFlag.Name, conf.privateKey))
	assert.NoError(tb, set.Set(outputPathFlag.Name, conf.outputPath))
	return cli.NewContext(&app, set, nil)
}

func createRandomKeystore(t testing.TB, password string) (*v2keymanager.Keystore, bls.SecretKey) {
	encryptor := keystorev4.New()
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	validatingKey := bls.RandKey()
	pubKey := validatingKey.PublicKey().Marshal()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	require.NoError(t, err)
	return &v2keymanager.Keystore{
		Crypto:  cryptoFields,
		Pubkey:  fmt.Sprintf("%x", pubKey),
		ID:      id.String(),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}, validatingKey
}

func setupRandomDir(t testing.TB) string {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err)
	randDir := filepath.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	require.NoError(t, os.MkdirAll(randDir, os.ModePerm))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(randDir), "Failed to remove directory")
	})
	return randDir
}

func TestDecrypt(t *testing.T) {
	keystoresDir := setupRandomDir(t)
	password := "secretPassw0rd$1999"
	keystore, privKey := createRandomKeystore(t, password)
	// We write a random keystore to a keystores directory.
	encodedKeystore, err := json.MarshalIndent(keystore, "", "\t")
	require.NoError(t, err)
	keystoreFilePath := filepath.Join(keystoresDir, "keystore.json")
	require.NoError(t, ioutil.WriteFile(
		keystoreFilePath, encodedKeystore, params.BeaconIoConfig().ReadWritePermissions),
	)

	cliCtx := setupCliContext(t, &cliConfig{
		keystoresPath: keystoreFilePath,
		password:      password,
	})

	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// We attempt to decrypt the keystore file we just wrote to disk.
	require.NoError(t, decrypt(cliCtx))

	require.NoError(t, w.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)

	// We capture output from stdout.
	os.Stdout = rescueStdout
	stringOutput := string(out)

	// We capture the results of stdout to check the public key and private keys
	// were both printed to stdout.
	assert.Equal(t, strings.Contains(stringOutput, keystore.Pubkey), true)
	assert.Equal(t, strings.Contains(stringOutput, fmt.Sprintf("%#x", privKey.Marshal())), true)
}

func TestEncrypt(t *testing.T) {
	keystoresDir := setupRandomDir(t)
	password := "secretPassw0rd$1999"
	keystoreFilePath := filepath.Join(keystoresDir, "keystore.json")
	privKey := bls.RandKey()

	cliCtx := setupCliContext(t, &cliConfig{
		outputPath: keystoreFilePath,
		password:   password,
		privateKey: fmt.Sprintf("%#x", privKey.Marshal()),
	})

	rescueStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// We attempt to encrypt the secret key and save it to the output path.
	require.NoError(t, encrypt(cliCtx))

	require.NoError(t, w.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)

	// We capture output from stdout.
	os.Stdout = rescueStdout
	stringOutput := string(out)

	// We capture the results of stdout to check the public key was printed to stdout.
	assert.Equal(
		t,
		strings.Contains(stringOutput, fmt.Sprintf("%x", privKey.PublicKey().Marshal())),
		true,
	)
}

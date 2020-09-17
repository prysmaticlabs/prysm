package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
)

type mockRemoteKeymanager struct {
	publicKeys [][48]byte
	opts       *remote.KeymanagerOpts
}

func (m *mockRemoteKeymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	return m.publicKeys, nil
}

func (m *mockRemoteKeymanager) Sign(context.Context, *validatorpb.SignRequest) (bls.Signature, error) {
	return nil, nil
}

func TestListAccounts_DirectKeymanager(t *testing.T) {
	walletDir, passwordsDir, walletPasswordFile := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     v2keymanager.Direct,
		walletPasswordFile: walletPasswordFile,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: v2keymanager.Direct,
			WalletPassword: "Passwordz0320$",
		},
	})
	require.NoError(t, err)
	keymanager, err := direct.NewKeymanager(
		cliCtx.Context,
		&direct.SetupConfig{
			Wallet: w,
			Opts:   direct.DefaultKeymanagerOpts(),
		},
	)
	require.NoError(t, err)

	numAccounts := 5
	for i := 0; i < numAccounts; i++ {
		_, err := keymanager.CreateAccount(cliCtx.Context)
		require.NoError(t, err)
	}
	rescueStdout := os.Stdout
	r, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer

	// We call the list direct keymanager accounts function.
	require.NoError(t, listDirectKeymanagerAccounts(context.Background(), true /* show deposit data */, keymanager))

	require.NoError(t, writer.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	os.Stdout = rescueStdout

	// Get stdout content and split to lines
	newLine := fmt.Sprintln()
	lines := strings.Split(string(out), newLine)

	// Expected output example:
	/*
		(keymanager kind) non-HD wallet

		Showing 5 validator accounts
		View the eth1 deposit transaction data for your accounts by running `validator accounts-v2 list --show-deposit-data

		Account 0 | briefly-intimate-condor
		[validating public key] 0x835980f321aa6c38cb4818f4e67ffd4e6b59e1e043f621ac8bb116dc32bbc9c950ff717edfa4d7b07b928ab072a2f9d2
		If you imported your account coming from the eth2 launchpad, you will find your deposit_data.json in the eth2.0-deposit-cli's validator_keys folder


		Account 1 | notably-premium-treefrog
		[validating public key] 0xb2e0179190c4270e8765ef2d38bfe89ea0d1be29ca6ba29212a2a34d4b286cae7571712593e0f30bc1645080afa3ce3c
		If you imported your account coming from the eth2 launchpad, you will find your deposit_data.json in the eth2.0-deposit-cli's validator_keys folder


		Account 2 | thankfully-sacred-hippo
		[validating public key] 0x87941b935932ac77aeb2b825e67cf442ebf0b958341e6d604b756025c047bc9cdbb6a46e9e35ed7f2f2980cacb8b3b56
		If you imported your account coming from the eth2 launchpad, you will find your deposit_data.json in the eth2.0-deposit-cli's validator_keys folder


		Account 3 | secondly-main-cod
		[validating public key] 0xa7dd8c42a0478f995a3dccd990211486482ddb67fa3da848e08a71f9966027ecb99a8f1c3bb84520ec85a2267598ee7e
		If you imported your account coming from the eth2 launchpad, you will find your deposit_data.json in the eth2.0-deposit-cli's validator_keys folder


		Account 4 | regularly-safe-ghoul
		[validating public key] 0x93833aeff64cd6861165e76615b7617d4ff87f33955970c37f7e7ddb3470ad040ef8508db744eef44c92f42928316e6f
		If you imported your account coming from the eth2 launchpad, you will find your deposit_data.json in the eth2.0-deposit-cli's validator_keys folder



	*/

	// Expected output format definition
	const prologLength = 4
	const accountLength = 5
	const epilogLength = 2
	const nameOffset = 1
	const keyOffset = 2

	// Require the output has correct number of lines
	lineCount := prologLength + accountLength*numAccounts + epilogLength
	require.Equal(t, lineCount, len(lines))

	// Assert the keymanager kind is printed on the first line.
	kindString := "non-HD"
	kindFound := strings.Contains(lines[0], kindString)
	assert.Equal(t, true, kindFound, "Keymanager Kind %s not found on the first line", kindString)

	// Get account names and require the correct count
	accountNames, err := keymanager.ValidatingAccountNames()
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(accountNames))

	// Assert that account names are printed on the correct lines
	for i, accountName := range accountNames {
		lineNumber := prologLength + accountLength*i + nameOffset
		accountNameFound := strings.Contains(lines[lineNumber], accountName)
		assert.Equal(t, true, accountNameFound, "Account Name %s not found on line number %d", accountName, lineNumber)
	}

	// Get public keys and require the correct count
	pubKeys, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(pubKeys))

	// Assert that public keys are printed on the correct lines
	for i, key := range pubKeys {
		lineNumber := prologLength + accountLength*i + keyOffset
		keyString := fmt.Sprintf("%#x", key)
		keyFound := strings.Contains(lines[lineNumber], keyString)
		assert.Equal(t, true, keyFound, "Public Key %s not found on line number %d", keyString, lineNumber)
	}
}

func TestListAccounts_DerivedKeymanager(t *testing.T) {
	walletDir, passwordsDir, passwordFilePath := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:          walletDir,
		passwordsDir:       passwordsDir,
		keymanagerKind:     v2keymanager.Derived,
		walletPasswordFile: passwordFilePath,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: v2keymanager.Derived,
			WalletPassword: "Passwordz0320$",
		},
	})
	require.NoError(t, err)

	keymanager, err := derived.NewKeymanager(
		cliCtx.Context,
		&derived.SetupConfig{
			Opts:                derived.DefaultKeymanagerOpts(),
			Wallet:              w,
			SkipMnemonicConfirm: true,
		},
	)
	require.NoError(t, err)

	numAccounts := 5
	depositDataForAccounts := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		_, err := keymanager.CreateAccount(cliCtx.Context, false /*logAccountInfo*/)
		require.NoError(t, err)
		enc, err := keymanager.DepositDataForAccount(uint64(i))
		require.NoError(t, err)
		depositDataForAccounts[i] = enc
	}

	rescueStdout := os.Stdout
	r, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer

	// We call the list direct keymanager accounts function.
	require.NoError(t, listDerivedKeymanagerAccounts(cliCtx.Context, true /* show deposit data */, keymanager))

	require.NoError(t, writer.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	os.Stdout = rescueStdout

	// Get stdout content and split to lines
	newLine := fmt.Sprintln()
	lines := strings.Split(string(out), newLine)

	// Expected output example:
	/*
		(keymanager kind) derived, (HD) hierarchical-deterministic
		(derivation format) m / purpose / coin_type / account_index / withdrawal_key / validating_key
		Showing 2 validator accounts

		Account 0 | uniformly-winning-gopher
		[withdrawal public key] 0xad416c4ba9a729921036563ceb66a556692a21613a1a09f19260292dd9573ee50cb0614087d65bd48fe645232d2ca2a6
		[derivation path] m/12381/3600/0/0
		[validating public key] 0xa9c2202dc0fdc74f00261947f325216246c34eafe76c327c5e2c88a5756c03a0f464b3b595af6511ffd212ecd45e07dd
		[derivation path] m/12381/3600/0/0/0

		======================Eth1 Deposit Transaction Data=====================

		0x22895118000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000120a3b144364879ec579415348630371a16ad52314ba7342a0635425512c7d9fdaa0000000000000000000000000000000000000000000000000000000000000030a9c2202dc0fdc74f00261947f325216246c34eafe76c327c5e2c88a5756c03a0f464b3b595af6511ffd212ecd45e07dd00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000fc991cc3a9babe8fbf430d819c7d79f6350927114fa7ce6593a97c235aca2a0000000000000000000000000000000000000000000000000000000000000060a96f65132f70bf053779bedc3a943c2f1ac6a96e896ca7159521f7663968bb4c79a5307eb22ff58f8de35749c824382617654a8f072efb24cd77739538bebaef1ea5b0943188ecbc5e1785feb48c4e6e9d129478e40bafba8d9f21140de5da55

		===================================================================

		Account 1 | curiously-diverse-rooster
		[withdrawal public key] 0x98d9bdf1fc2fc7c237d903e92c38e30f049412fba879923d9f6400fb8f0bc57e43fc5406917f66a445f7a0ab3214ac28
		[derivation path] m/12381/3600/1/0
		[validating public key] 0x849d5d2802d17cda4e9e79a7ca98534a5b35e165631ec8325b5cdcea0ec2a83c2cc1955ca3cc0c16a3b4f051473f471d
		[derivation path] m/12381/3600/1/0/0

		======================Eth1 Deposit Transaction Data=====================

		0x22895118000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000012027dc7cc27483f42ee9f98b9a48105239de731db98d81f32dfe690d221f4fec340000000000000000000000000000000000000000000000000000000000000030849d5d2802d17cda4e9e79a7ca98534a5b35e165631ec8325b5cdcea0ec2a83c2cc1955ca3cc0c16a3b4f051473f471d00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000007c25d0d2051e8c77b9e2f0eb8184df4ca561f6597ad9029cb5a963bedf29000000000000000000000000000000000000000000000000000000000000006090fa0b1263164a0d4b3db35c6a4af0cfef19faf722fc44de263bc5a1a1ac8c0f2e8e7b18964798e9bc7a494f82cf735b0d5a7049e6bc0224c8a1000ef33c67e136d49560668d5ae6e95816997d9ed5165ee8ed5be02df63a9c88239e33551787

		===================================================================

	*/

	// Expected output format definition
	const prologLength = 3
	const accountLength = 12
	const epilogLength = 1
	const nameOffset = 1
	const keyOffset = 4
	const depositOffset = 9

	// Require the output has correct number of lines
	lineCount := prologLength + accountLength*numAccounts + epilogLength
	require.Equal(t, lineCount, len(lines))

	// Assert the keymanager kind is printed on the first line.
	kindString := w.KeymanagerKind().String()
	kindFound := strings.Contains(lines[0], kindString)
	assert.Equal(t, true, kindFound, "Keymanager Kind %s not found on the first line", kindString)

	// Get account names and require the correct count
	accountNames, err := keymanager.ValidatingAccountNames(cliCtx.Context)
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(accountNames))

	// Assert that account names are printed on the correct lines
	for i, accountName := range accountNames {
		lineNumber := prologLength + accountLength*i + nameOffset
		accountNameFound := strings.Contains(lines[lineNumber], accountName)
		assert.Equal(t, true, accountNameFound, "Account Name %s not found on line number %d", accountName, lineNumber)
	}

	// Get public keys and require the correct count
	pubKeys, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	require.NoError(t, err)
	require.Equal(t, numAccounts, len(pubKeys))

	// Assert that public keys are printed on the correct lines
	for i, key := range pubKeys {
		lineNumber := prologLength + accountLength*i + keyOffset
		keyString := fmt.Sprintf("%#x", key)
		keyFound := strings.Contains(lines[lineNumber], keyString)
		assert.Equal(t, true, keyFound, "Public Key %s not found on line number %d", keyString, lineNumber)
	}

	// Assert that deposit data are printed on the correct lines
	for i, deposit := range depositDataForAccounts {
		lineNumber := prologLength + accountLength*i + depositOffset
		depositString := fmt.Sprintf("%#x", deposit)
		depositFound := strings.Contains(lines[lineNumber], depositString)
		assert.Equal(t, true, depositFound, "Deposit data %s not found on line number %d", depositString, lineNumber)
	}
}

func TestListAccounts_RemoteKeymanager(t *testing.T) {
	walletDir, _, _ := setupWalletAndPasswordsDir(t)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:      walletDir,
		keymanagerKind: v2keymanager.Remote,
	})
	w, err := CreateWalletWithKeymanager(cliCtx.Context, &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: v2keymanager.Remote,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)

	rescueStdout := os.Stdout
	r, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer

	numAccounts := 3
	pubKeys := make([][48]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		key := make([]byte, 48)
		copy(key, strconv.Itoa(i))
		pubKeys[i] = bytesutil.ToBytes48(key)
	}
	km := &mockRemoteKeymanager{
		publicKeys: pubKeys,
		opts: &remote.KeymanagerOpts{
			RemoteCertificate: &remote.CertificateConfig{
				ClientCertPath: "/tmp/client.crt",
				ClientKeyPath:  "/tmp/client.key",
				CACertPath:     "/tmp/ca.crt",
			},
			RemoteAddr: "localhost:4000",
		},
	}
	// We call the list remote keymanager accounts function.
	require.NoError(t, listRemoteKeymanagerAccounts(context.Background(), w, km, km.opts))

	require.NoError(t, writer.Close())
	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	os.Stdout = rescueStdout

	// Get stdout content and split to lines
	newLine := fmt.Sprintln()
	lines := strings.Split(string(out), newLine)

	// Expected output example:
	/*
		(keymanager kind) remote signer
		(configuration file path) /tmp/79336/wallet/remote/keymanageropts.json

		Configuration options
		Remote gRPC address: localhost:4000
		Client cert path: /tmp/client.crt
		Client key path: /tmp/client.key
		CA cert path: /tmp/ca.crt

		Showing 3 validator accounts

		equally-primary-foal
		[validating public key] 0x300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000


		rationally-charmed-werewolf
		[validating public key] 0x310000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000


	*/

	// Expected output format definition
	const prologLength = 10
	const configOffset = 4
	const configLength = 4
	const accountLength = 4
	const nameOffset = 1
	const keyOffset = 2
	const epilogLength = 1

	// Require the output has correct number of lines
	lineCount := prologLength + accountLength*numAccounts + epilogLength
	require.Equal(t, lineCount, len(lines))

	// Assert the keymanager kind is printed on the first line.
	kindString := w.KeymanagerKind().String()
	kindFound := strings.Contains(lines[0], kindString)
	assert.Equal(t, true, kindFound, "Keymanager Kind %s not found on the first line", kindString)

	// Assert that Configuration is printed in the right position
	configLines := lines[configOffset:(configOffset + configLength)]
	configExpected := km.opts.String()
	configActual := fmt.Sprintln(strings.Join(configLines, newLine))
	assert.Equal(t, configExpected, configActual, "Configuration not found at the expected position")

	// Assert that account names are printed on the correct lines
	for i := 0; i < numAccounts; i++ {
		lineNumber := prologLength + accountLength*i + nameOffset
		accountName := petnames.DeterministicName(pubKeys[i][:], "-")
		accountNameFound := strings.Contains(lines[lineNumber], accountName)
		assert.Equal(t, true, accountNameFound, "Account Name %s not found on line number %d", accountName, lineNumber)
	}

	// Assert that public keys are printed on the correct lines
	for i, key := range pubKeys {
		lineNumber := prologLength + accountLength*i + keyOffset
		keyString := fmt.Sprintf("%#x", key)
		keyFound := strings.Contains(lines[lineNumber], keyString)
		assert.Equal(t, true, keyFound, "Public Key %s not found on line number %d", keyString, lineNumber)
	}
}

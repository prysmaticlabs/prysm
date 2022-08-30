package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	validatorserviceconfig "github.com/prysmaticlabs/prysm/v3/config/validator/service"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbservice "github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/iface"
	mock "github.com/prysmaticlabs/prysm/v3/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/client"
	"github.com/prysmaticlabs/prysm/v3/validator/db/kv"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/derived"
	remoteweb3signer "github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote-web3signer"
	"github.com/prysmaticlabs/prysm/v3/validator/slashing-protection-history/format"
	mocks "github.com/prysmaticlabs/prysm/v3/validator/testing"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestServer_ListKeystores(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.ErrorContains(t, "Prysm Wallet not initialized. Please create a new wallet.", err)
	})
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	numAccounts := 50
	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err = dr.RecoverAccountsFromMnemonic(ctx, mocks.TestMnemonic, "", numAccounts)
	require.NoError(t, err)
	expectedKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	t.Run("returns proper data with existing keystores", func(t *testing.T) {
		resp, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.NoError(t, err)
		require.Equal(t, numAccounts, len(resp.Data))
		for i := 0; i < numAccounts; i++ {
			require.DeepEqual(t, expectedKeys[i][:], resp.Data[i].ValidatingPubkey)
			require.Equal(
				t,
				fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i),
				resp.Data[i].DerivationPath,
			)
		}
	})
}

func TestServer_ImportKeystores(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		response, err := s.ImportKeystores(context.Background(), &ethpbservice.ImportKeystoresRequest{})
		require.NoError(t, err)
		require.Equal(t, 0, len(response.Data))
	})
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	t.Run("200 response even if faulty keystore in request", func(t *testing.T) {
		response, err := s.ImportKeystores(context.Background(), &ethpbservice.ImportKeystoresRequest{
			Keystores: []string{"hi"},
			Passwords: []string{"hi"},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(response.Data))
		require.Equal(t, ethpbservice.ImportedKeystoreStatus_ERROR, response.Data[0].Status)
	})
	t.Run("200 response even if  no passwords in request", func(t *testing.T) {
		response, err := s.ImportKeystores(context.Background(), &ethpbservice.ImportKeystoresRequest{
			Keystores: []string{"hi"},
			Passwords: []string{},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(response.Data))
		require.Equal(t, ethpbservice.ImportedKeystoreStatus_ERROR, response.Data[0].Status)
	})
	t.Run("200 response even if  keystores more than passwords in request", func(t *testing.T) {
		response, err := s.ImportKeystores(context.Background(), &ethpbservice.ImportKeystoresRequest{
			Keystores: []string{"hi", "hi"},
			Passwords: []string{"hi"},
		})
		require.NoError(t, err)
		require.Equal(t, 2, len(response.Data))
		require.Equal(t, ethpbservice.ImportedKeystoreStatus_ERROR, response.Data[0].Status)
	})
	t.Run("200 response even if number of passwords does not match number of keystores", func(t *testing.T) {
		response, err := s.ImportKeystores(context.Background(), &ethpbservice.ImportKeystoresRequest{
			Keystores: []string{"hi"},
			Passwords: []string{"hi", "hi"},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(response.Data))
		require.Equal(t, ethpbservice.ImportedKeystoreStatus_ERROR, response.Data[0].Status)
	})
	t.Run("200 response even if faulty slashing protection data", func(t *testing.T) {
		numKeystores := 5
		password := "12345678"
		encodedKeystores := make([]string, numKeystores)
		passwords := make([]string, numKeystores)
		for i := 0; i < numKeystores; i++ {
			enc, err := json.Marshal(createRandomKeystore(t, password))
			encodedKeystores[i] = string(enc)
			require.NoError(t, err)
			passwords[i] = password
		}
		resp, err := s.ImportKeystores(context.Background(), &ethpbservice.ImportKeystoresRequest{
			Keystores:          encodedKeystores,
			Passwords:          passwords,
			SlashingProtection: "foobar",
		})
		require.NoError(t, err)
		require.Equal(t, numKeystores, len(resp.Data))
		for _, st := range resp.Data {
			require.Equal(t, ethpbservice.ImportedKeystoreStatus_ERROR, st.Status)
		}
	})
	t.Run("returns proper statuses for keystores in request", func(t *testing.T) {
		numKeystores := 5
		password := "12345678"
		keystores := make([]*keymanager.Keystore, numKeystores)
		passwords := make([]string, numKeystores)
		publicKeys := make([][fieldparams.BLSPubkeyLength]byte, numKeystores)
		for i := 0; i < numKeystores; i++ {
			keystores[i] = createRandomKeystore(t, password)
			pubKey, err := hex.DecodeString(keystores[i].Pubkey)
			require.NoError(t, err)
			publicKeys[i] = bytesutil.ToBytes48(pubKey)
			passwords[i] = password
		}

		// Create a validator database.
		validatorDB, err := kv.NewKVStore(ctx, defaultWalletPath, &kv.Config{
			PubKeys: publicKeys,
		})
		require.NoError(t, err)
		s.valDB = validatorDB

		// Have to close it after import is done otherwise it complains db is not open.
		defer func() {
			require.NoError(t, validatorDB.Close())
		}()
		encodedKeystores := make([]string, numKeystores)
		for i := 0; i < numKeystores; i++ {
			enc, err := json.Marshal(keystores[i])
			require.NoError(t, err)
			encodedKeystores[i] = string(enc)
		}

		// Generate mock slashing history.
		attestingHistory := make([][]*kv.AttestationRecord, 0)
		proposalHistory := make([]kv.ProposalHistoryForPubkey, len(publicKeys))
		for i := 0; i < len(publicKeys); i++ {
			proposalHistory[i].Proposals = make([]kv.Proposal, 0)
		}
		mockJSON, err := mocks.MockSlashingProtectionJSON(publicKeys, attestingHistory, proposalHistory)
		require.NoError(t, err)

		// JSON encode the protection JSON and save it.
		encodedSlashingProtection, err := json.Marshal(mockJSON)
		require.NoError(t, err)

		resp, err := s.ImportKeystores(context.Background(), &ethpbservice.ImportKeystoresRequest{
			Keystores:          encodedKeystores,
			Passwords:          passwords,
			SlashingProtection: string(encodedSlashingProtection),
		})
		require.NoError(t, err)
		require.Equal(t, numKeystores, len(resp.Data))
		for _, status := range resp.Data {
			require.Equal(t, ethpbservice.ImportedKeystoreStatus_IMPORTED, status.Status)
		}
	})
}

func TestServer_ImportKeystores_WrongKeymanagerKind(t *testing.T) {
	ctx := context.Background()
	w := wallet.NewWalletForWeb3Signer()
	root := make([]byte, fieldparams.RootLength)
	root[0] = 1
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false, Web3SignerConfig: &remoteweb3signer.SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		PublicKeysURL:         "http://example.com/public_keys",
	}})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	response, err := s.ImportKeystores(context.Background(), &ethpbservice.ImportKeystoresRequest{
		Keystores: []string{"hi"},
		Passwords: []string{"hi"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(response.Data))
	require.Equal(t, ethpbservice.ImportedKeystoreStatus_ERROR, response.Data[0].Status)
	require.Equal(t, "Keymanager kind cannot import keys", response.Data[0].Message)
}

func TestServer_DeleteKeystores(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		response, err := s.DeleteKeystores(context.Background(), &ethpbservice.DeleteKeystoresRequest{})
		require.NoError(t, err)
		require.Equal(t, 0, len(response.Data))
	})
	ctx := context.Background()
	srv := setupServerWithWallet(t)

	// We recover 3 accounts from a test mnemonic.
	numAccounts := 3
	km, er := srv.validatorService.Keymanager()
	require.NoError(t, er)
	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err := dr.RecoverAccountsFromMnemonic(ctx, mocks.TestMnemonic, "", numAccounts)
	require.NoError(t, err)
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	// Create a validator database.
	validatorDB, err := kv.NewKVStore(ctx, defaultWalletPath, &kv.Config{
		PubKeys: publicKeys,
	})
	require.NoError(t, err)
	srv.valDB = validatorDB

	// Have to close it after import is done otherwise it complains db is not open.
	defer func() {
		require.NoError(t, validatorDB.Close())
	}()

	// Generate mock slashing history.
	attestingHistory := make([][]*kv.AttestationRecord, 0)
	proposalHistory := make([]kv.ProposalHistoryForPubkey, len(publicKeys))
	for i := 0; i < len(publicKeys); i++ {
		proposalHistory[i].Proposals = make([]kv.Proposal, 0)
	}
	mockJSON, err := mocks.MockSlashingProtectionJSON(publicKeys, attestingHistory, proposalHistory)
	require.NoError(t, err)

	// JSON encode the protection JSON and save it.
	encoded, err := json.Marshal(mockJSON)
	require.NoError(t, err)

	_, err = srv.ImportSlashingProtection(ctx, &validatorpb.ImportSlashingProtectionRequest{
		SlashingProtectionJson: string(encoded),
	})
	require.NoError(t, err)

	t.Run("no slashing protection response if no keys in request even if we have a history in DB", func(t *testing.T) {
		resp, err := srv.DeleteKeystores(context.Background(), &ethpbservice.DeleteKeystoresRequest{
			Pubkeys: nil,
		})
		require.NoError(t, err)
		require.Equal(t, "", resp.SlashingProtection)
	})

	// For ease of test setup, we'll give each public key a string identifier.
	publicKeysWithId := map[string][fieldparams.BLSPubkeyLength]byte{
		"a": publicKeys[0],
		"b": publicKeys[1],
		"c": publicKeys[2],
	}

	type keyCase struct {
		id                 string
		wantProtectionData bool
	}
	tests := []struct {
		keys         []*keyCase
		wantStatuses []ethpbservice.DeletedKeystoreStatus_Status
	}{
		{
			keys: []*keyCase{
				{id: "a", wantProtectionData: true},
				{id: "a", wantProtectionData: true},
				{id: "d"},
				{id: "c", wantProtectionData: true},
			},
			wantStatuses: []ethpbservice.DeletedKeystoreStatus_Status{
				ethpbservice.DeletedKeystoreStatus_DELETED,
				ethpbservice.DeletedKeystoreStatus_NOT_ACTIVE,
				ethpbservice.DeletedKeystoreStatus_NOT_FOUND,
				ethpbservice.DeletedKeystoreStatus_DELETED,
			},
		},
		{
			keys: []*keyCase{
				{id: "a", wantProtectionData: true},
				{id: "c", wantProtectionData: true},
			},
			wantStatuses: []ethpbservice.DeletedKeystoreStatus_Status{
				ethpbservice.DeletedKeystoreStatus_NOT_ACTIVE,
				ethpbservice.DeletedKeystoreStatus_NOT_ACTIVE,
			},
		},
		{
			keys: []*keyCase{
				{id: "x"},
			},
			wantStatuses: []ethpbservice.DeletedKeystoreStatus_Status{
				ethpbservice.DeletedKeystoreStatus_NOT_FOUND,
			},
		},
	}
	for _, tc := range tests {
		keys := make([][]byte, len(tc.keys))
		for i := 0; i < len(tc.keys); i++ {
			pk := publicKeysWithId[tc.keys[i].id]
			keys[i] = pk[:]
		}
		resp, err := srv.DeleteKeystores(ctx, &ethpbservice.DeleteKeystoresRequest{Pubkeys: keys})
		require.NoError(t, err)
		require.Equal(t, len(keys), len(resp.Data))
		slashingProtectionData := &format.EIPSlashingProtectionFormat{}
		require.NoError(t, json.Unmarshal([]byte(resp.SlashingProtection), slashingProtectionData))
		require.Equal(t, true, len(slashingProtectionData.Data) > 0)

		for i := 0; i < len(tc.keys); i++ {
			require.Equal(
				t,
				tc.wantStatuses[i],
				resp.Data[i].Status,
				fmt.Sprintf("Checking status for key %s", tc.keys[i].id),
			)
			if tc.keys[i].wantProtectionData {
				// We check that we can find the key in the slashing protection data.
				var found bool
				for _, dt := range slashingProtectionData.Data {
					if dt.Pubkey == fmt.Sprintf("%#x", keys[i]) {
						found = true
						break
					}
				}
				require.Equal(t, true, found)
			}
		}
	}
}

func TestServer_DeleteKeystores_FailedSlashingProtectionExport(t *testing.T) {
	ctx := context.Background()
	srv := setupServerWithWallet(t)

	// We recover 3 accounts from a test mnemonic.
	numAccounts := 3
	km, er := srv.validatorService.Keymanager()
	require.NoError(t, er)
	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err := dr.RecoverAccountsFromMnemonic(ctx, mocks.TestMnemonic, "", numAccounts)
	require.NoError(t, err)
	publicKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	// Create a validator database.
	validatorDB, err := kv.NewKVStore(ctx, defaultWalletPath, &kv.Config{
		PubKeys: publicKeys,
	})
	require.NoError(t, err)
	err = validatorDB.SaveGenesisValidatorsRoot(ctx, make([]byte, fieldparams.RootLength))
	require.NoError(t, err)
	srv.valDB = validatorDB

	// Have to close it after import is done otherwise it complains db is not open.
	defer func() {
		require.NoError(t, validatorDB.Close())
	}()

	response, err := srv.DeleteKeystores(context.Background(), &ethpbservice.DeleteKeystoresRequest{
		Pubkeys: [][]byte{[]byte("a")},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(response.Data))
	require.Equal(t, ethpbservice.DeletedKeystoreStatus_ERROR, response.Data[0].Status)
	require.Equal(t, "Non duplicate keys that were existing were deleted, but could not export slashing protection history.",
		response.Data[0].Message,
	)
}

func TestServer_DeleteKeystores_WrongKeymanagerKind(t *testing.T) {
	ctx := context.Background()
	w := wallet.NewWalletForWeb3Signer()
	root := make([]byte, fieldparams.RootLength)
	root[0] = 1
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false,
		Web3SignerConfig: &remoteweb3signer.SetupConfig{
			BaseEndpoint:          "http://example.com",
			GenesisValidatorsRoot: root,
			PublicKeysURL:         "http://example.com/public_keys",
		}})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	_, err = s.DeleteKeystores(ctx, &ethpbservice.DeleteKeystoresRequest{Pubkeys: [][]byte{[]byte("a")}})
	require.ErrorContains(t, "Wrong wallet type", err)
	require.ErrorContains(t, "Only Imported or Derived wallets can delete accounts", err)
}

func setupServerWithWallet(t testing.TB) *Server {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	w, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      defaultWalletPath,
			KeymanagerKind: keymanager.Derived,
			WalletPassword: strongPass,
		},
		SkipMnemonicConfirm: true,
	})
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
	})
	require.NoError(t, err)

	return &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
}

func createRandomKeystore(t testing.TB, password string) *keymanager.Keystore {
	encryptor := keystorev4.New()
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	validatingKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := validatingKey.PublicKey().Marshal()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	require.NoError(t, err)
	return &keymanager.Keystore{
		Crypto:  cryptoFields,
		Pubkey:  fmt.Sprintf("%x", pubKey),
		ID:      id.String(),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
}

func TestServer_ListRemoteKeys(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.ErrorContains(t, "Prysm Wallet not initialized. Please create a new wallet.", err)
	})
	ctx := context.Background()
	w := wallet.NewWalletForWeb3Signer()
	root := make([]byte, fieldparams.RootLength)
	root[0] = 1
	bytevalue, err := hexutil.Decode("0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a")
	require.NoError(t, err)
	pubkeys := [][fieldparams.BLSPubkeyLength]byte{bytesutil.ToBytes48(bytevalue)}
	config := &remoteweb3signer.SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		ProvidedPublicKeys:    pubkeys,
	}
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false, Web3SignerConfig: config})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
		Web3SignerConfig: config,
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	expectedKeys, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	t.Run("returns proper data with existing pub keystores", func(t *testing.T) {
		resp, err := s.ListRemoteKeys(context.Background(), &empty.Empty{})
		require.NoError(t, err)
		for i := 0; i < len(resp.Data); i++ {
			require.DeepEqual(t, expectedKeys[i][:], resp.Data[i].Pubkey)
		}
	})
}

func TestServer_ImportRemoteKeys(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.ErrorContains(t, "Prysm Wallet not initialized. Please create a new wallet.", err)
	})
	ctx := context.Background()
	w := wallet.NewWalletForWeb3Signer()
	root := make([]byte, fieldparams.RootLength)
	root[0] = 1
	config := &remoteweb3signer.SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		ProvidedPublicKeys:    nil,
	}
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false, Web3SignerConfig: config})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
		Web3SignerConfig: config,
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	bytevalue, err := hexutil.Decode("0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a")
	require.NoError(t, err)
	remoteKeys := []*ethpbservice.ImportRemoteKeysRequest_Keystore{
		{
			Pubkey: bytevalue,
		},
	}

	t.Run("returns proper data with existing pub keystores", func(t *testing.T) {
		resp, err := s.ImportRemoteKeys(context.Background(), &ethpbservice.ImportRemoteKeysRequest{
			RemoteKeys: remoteKeys,
		})
		expectedStatuses := []*ethpbservice.ImportedRemoteKeysStatus{
			{
				Status:  ethpbservice.ImportedRemoteKeysStatus_IMPORTED,
				Message: fmt.Sprintf("Successfully added pubkey: %v", hexutil.Encode(bytevalue)),
			},
		}
		require.NoError(t, err)
		for i := 0; i < len(resp.Data); i++ {
			require.DeepEqual(t, expectedStatuses[i], resp.Data[i])
		}
	})
}

func TestServer_DeleteRemoteKeys(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.ErrorContains(t, "Prysm Wallet not initialized. Please create a new wallet.", err)
	})
	ctx := context.Background()
	w := wallet.NewWalletForWeb3Signer()
	root := make([]byte, fieldparams.RootLength)
	root[0] = 1
	bytevalue, err := hexutil.Decode("0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a")
	require.NoError(t, err)
	pubkeys := [][fieldparams.BLSPubkeyLength]byte{bytesutil.ToBytes48(bytevalue)}
	config := &remoteweb3signer.SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		ProvidedPublicKeys:    pubkeys,
	}
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false, Web3SignerConfig: config})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.MockValidator{
			Km: km,
		},
		Web3SignerConfig: config,
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}

	t.Run("returns proper data with existing pub keystores", func(t *testing.T) {
		resp, err := s.DeleteRemoteKeys(context.Background(), &ethpbservice.DeleteRemoteKeysRequest{
			Pubkeys: [][]byte{bytevalue},
		})
		expectedStatuses := []*ethpbservice.DeletedRemoteKeysStatus{
			{
				Status:  ethpbservice.DeletedRemoteKeysStatus_DELETED,
				Message: fmt.Sprintf("Successfully deleted pubkey: %v", hexutil.Encode(bytevalue)),
			},
		}
		require.NoError(t, err)
		for i := 0; i < len(resp.Data); i++ {
			require.DeepEqual(t, expectedStatuses[i], resp.Data[i])

		}
		expectedKeys, err := km.FetchValidatingPublicKeys(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, len(expectedKeys))
	})
}

func TestServer_ListFeeRecipientByPubkey(t *testing.T) {
	ctx := context.Background()
	byteval, err := hexutil.Decode("0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	require.NoError(t, err)

	type want struct {
		EthAddress string
	}

	tests := []struct {
		name    string
		args    *validatorserviceconfig.ProposerSettings
		want    *want
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): {
						FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9"),
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9"),
				},
			},
			want: &want{
				EthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
			wantErr: false,
		},
		{
			name: "empty settings",
			args: nil,
			want: &want{
				EthAddress: params.BeaconConfig().DefaultFeeRecipient.Hex(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, err := client.NewValidatorService(ctx, &client.Config{
				Validator:        &mock.MockValidator{},
				ProposerSettings: tt.args,
			})
			require.NoError(t, err)
			s := &Server{
				validatorService: vs,
			}
			got, err := s.ListFeeRecipientByPubkey(ctx, &ethpbservice.PubkeyRequest{Pubkey: byteval})
			require.NoError(t, err)
			assert.Equal(t, tt.want.EthAddress, common.BytesToAddress(got.Data.Ethaddress).Hex())
		})
	}
}
func TestServer_SetFeeRecipientByPubkey(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	byteval, err := hexutil.Decode("0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	require.NoError(t, err)
	type want struct {
		EthAddress string
	}
	tests := []struct {
		name             string
		args             string
		proposerSettings *validatorserviceconfig.ProposerSettings
		want             *want
		wantErr          bool
	}{
		{
			name: "Happy Path Test",
			args: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			want: &want{
				EthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, err := client.NewValidatorService(ctx, &client.Config{
				Validator:        &mock.MockValidator{},
				ProposerSettings: tt.proposerSettings,
			})
			require.NoError(t, err)
			s := &Server{
				validatorService: vs,
			}
			_, err = s.SetFeeRecipientByPubkey(ctx, &ethpbservice.SetFeeRecipientByPubkeyRequest{Pubkey: byteval, Ethaddress: common.HexToAddress(tt.args).Bytes()})
			require.NoError(t, err)
			assert.Equal(t, tt.want.EthAddress, s.validatorService.ProposerSettings.ProposeConfig[bytesutil.ToBytes48(byteval)].FeeRecipient.Hex())
		})
	}
}

func TestServer_DeleteFeeRecipientByPubkey(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	byteval, err := hexutil.Decode("0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	require.NoError(t, err)
	type want struct {
		EthAddress string
	}
	tests := []struct {
		name             string
		proposerSettings *validatorserviceconfig.ProposerSettings
		want             *want
		wantErr          bool
	}{
		{
			name: "Happy Path Test",
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): {
						FeeRecipient: common.HexToAddress("0x055Fb65722E7b2455012BFEBf6177F1D2e9738D5"),
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9"),
				},
			},
			want: &want{
				EthAddress: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9").Hex(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, err := client.NewValidatorService(ctx, &client.Config{
				Validator:        &mock.MockValidator{},
				ProposerSettings: tt.proposerSettings,
			})
			require.NoError(t, err)
			s := &Server{
				validatorService: vs,
			}
			_, err = s.DeleteFeeRecipientByPubkey(ctx, &ethpbservice.PubkeyRequest{Pubkey: byteval})
			require.NoError(t, err)
			assert.Equal(t, tt.want.EthAddress, s.validatorService.ProposerSettings.ProposeConfig[bytesutil.ToBytes48(byteval)].FeeRecipient.Hex())
		})
	}
}

func TestServer_GetGasLimit(t *testing.T) {
	ctx := context.Background()
	byteval, err := hexutil.Decode("0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	byteval2, err2 := hexutil.Decode("0x1234567878903438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	require.NoError(t, err)
	require.NoError(t, err2)

	tests := []struct {
		name   string
		args   *validatorserviceconfig.ProposerSettings
		pubkey [48]byte
		want   uint64
	}{
		{
			name: "ProposerSetting for specific pubkey exists",
			args: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 123456789},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 987654321},
				},
			},
			pubkey: bytesutil.ToBytes48(byteval),
			want:   123456789,
		},
		{
			name: "ProposerSetting for specific pubkey does not exist",
			args: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 123456789},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 987654321},
				},
			},
			// no settings for the following validator, so the gaslimit returned is the default value.
			pubkey: bytesutil.ToBytes48(byteval2),
			want:   987654321,
		},
		{
			name:   "No proposerSetting at all",
			args:   nil,
			pubkey: bytesutil.ToBytes48(byteval),
			want:   params.BeaconConfig().DefaultBuilderGasLimit,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, err := client.NewValidatorService(ctx, &client.Config{
				Validator:        &mock.MockValidator{},
				ProposerSettings: tt.args,
			})
			require.NoError(t, err)
			s := &Server{
				validatorService: vs,
			}
			got, err := s.GetGasLimit(ctx, &ethpbservice.PubkeyRequest{Pubkey: tt.pubkey[:]})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Data.GasLimit)
		})
	}
}

func TestServer_SetGasLimit(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	pubkey1, err := hexutil.Decode("0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	pubkey2, err2 := hexutil.Decode("0xbedefeaa94e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2cdddddddddddddddddddddddd")
	require.NoError(t, err)
	require.NoError(t, err2)

	type want struct {
		pubkey   []byte
		gaslimit uint64
	}

	tests := []struct {
		name             string
		pubkey           []byte
		newGasLimit      uint64
		proposerSettings *validatorserviceconfig.ProposerSettings
		w                []want
	}{
		{
			name:        "update existing gas limit",
			pubkey:      pubkey1,
			newGasLimit: 9999,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 123456789},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 987654321},
				},
			},
			w: []want{
				want{
					pubkey:   pubkey1,
					gaslimit: 9999,
				},
			},
		},
		{
			name:        "insert a new gas limit",
			pubkey:      pubkey2,
			newGasLimit: 8888,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 123456789},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 987654321},
				},
			},
			w: []want{
				want{
					pubkey:   pubkey1,
					gaslimit: 123456789,
				},
				want{
					pubkey:   pubkey2,
					gaslimit: 8888,
				},
			},
		},
		{
			name:        "create new gas limit value for nil ProposerSettings.ProposeConfig",
			pubkey:      pubkey1,
			newGasLimit: 8888,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 987654321},
				},
			},
			w: []want{
				want{
					pubkey:   pubkey1,
					gaslimit: 8888,
				},
			},
		},
		{
			name:        "create new gas limit value for nil proposerSettings",
			pubkey:      pubkey1,
			newGasLimit: 7777,
			// proposerSettings is not set - we need to create proposerSettings and set gaslimit properly
			w: []want{
				want{
					pubkey:   pubkey1,
					gaslimit: 7777,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, err := client.NewValidatorService(ctx, &client.Config{
				Validator:        &mock.MockValidator{},
				ProposerSettings: tt.proposerSettings,
			})
			require.NoError(t, err)
			s := &Server{
				validatorService: vs,
			}
			_, err = s.SetGasLimit(ctx, &ethpbservice.SetGasLimitRequest{Pubkey: tt.pubkey, GasLimit: tt.newGasLimit})
			require.NoError(t, err)
			for _, w := range tt.w {
				assert.Equal(t, w.gaslimit, uint64(s.validatorService.ProposerSettings.ProposeConfig[bytesutil.ToBytes48(w.pubkey)].BuilderConfig.GasLimit))
			}
		})
	}
}

func TestServer_DeleteGasLimit(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	pubkey1, err := hexutil.Decode("0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	pubkey2, err2 := hexutil.Decode("0xbedefeaa94e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2cdddddddddddddddddddddddd")
	require.NoError(t, err)
	require.NoError(t, err2)

	// This test changes global default values, we do not want this to side-affect other
	// tests, so store the origin global default and then restore after tests are done.
	originBeaconChainGasLimit := params.BeaconConfig().DefaultBuilderGasLimit
	defer func() {
		params.BeaconConfig().DefaultBuilderGasLimit = originBeaconChainGasLimit
	}()

	globalDefaultGasLimit := validatorserviceconfig.Uint64(0xbbdd)

	type want struct {
		pubkey   []byte
		gaslimit validatorserviceconfig.Uint64
	}

	tests := []struct {
		name             string
		pubkey           []byte
		proposerSettings *validatorserviceconfig.ProposerSettings
		wantError        error
		w                []want
	}{
		{
			name:   "delete existing gas limit with default config",
			pubkey: pubkey1,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validatorserviceconfig.Uint64(987654321)},
					},
					bytesutil.ToBytes48(pubkey2): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validatorserviceconfig.Uint64(123456789)},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validatorserviceconfig.Uint64(5555)},
				},
			},
			wantError: nil,
			w: []want{
				{
					pubkey: pubkey1,
					// After deletion, use DefaultConfig.BuilderConfig.GasLimit.
					gaslimit: validatorserviceconfig.Uint64(5555),
				},
				{
					pubkey:   pubkey2,
					gaslimit: validatorserviceconfig.Uint64(123456789),
				},
			},
		},
		{
			name:   "delete existing gas limit with no default config",
			pubkey: pubkey1,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validatorserviceconfig.Uint64(987654321)},
					},
					bytesutil.ToBytes48(pubkey2): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validatorserviceconfig.Uint64(123456789)},
					},
				},
			},
			wantError: nil,
			w: []want{
				{
					pubkey: pubkey1,
					// After deletion, use global default, because DefaultConfig is not set at all.
					gaslimit: globalDefaultGasLimit,
				},
				{
					pubkey:   pubkey2,
					gaslimit: validatorserviceconfig.Uint64(123456789),
				},
			},
		},
		{
			name:   "delete nonexist gas limit",
			pubkey: pubkey2,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validatorserviceconfig.Uint64(987654321)},
					},
				},
			},
			wantError: fmt.Errorf("%s", codes.NotFound.String()),
			w: []want{
				// pubkey1's gaslimit is unaffected
				{
					pubkey:   pubkey1,
					gaslimit: validatorserviceconfig.Uint64(987654321),
				},
			},
		},
		{
			name:      "delete nonexist gas limit 2",
			pubkey:    pubkey2,
			wantError: fmt.Errorf("%s", codes.NotFound.String()),
			w:         []want{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, err := client.NewValidatorService(ctx, &client.Config{
				Validator:        &mock.MockValidator{},
				ProposerSettings: tt.proposerSettings,
			})
			require.NoError(t, err)
			s := &Server{
				validatorService: vs,
			}
			// Set up global default value for builder gas limit.
			params.BeaconConfig().DefaultBuilderGasLimit = uint64(globalDefaultGasLimit)
			_, err = s.DeleteGasLimit(ctx, &ethpbservice.DeleteGasLimitRequest{Pubkey: tt.pubkey})
			if tt.wantError != nil {
				assert.ErrorContains(t, fmt.Sprintf("code = %s", tt.wantError.Error()), err)
			} else {
				require.NoError(t, err)
			}
			for _, w := range tt.w {
				assert.Equal(t, w.gaslimit, s.validatorService.ProposerSettings.ProposeConfig[bytesutil.ToBytes48(w.pubkey)].BuilderConfig.GasLimit)
			}
		})
	}
}

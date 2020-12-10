package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	wallet2 "github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	dbTest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	logTest "github.com/sirupsen/logrus/hooks/test"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
	"google.golang.org/grpc/metadata"
)

var _ shared.Service = (*ValidatorService)(nil)
var _ BeaconNodeInfoFetcher = (*ValidatorService)(nil)
var _ GenesisFetcher = (*ValidatorService)(nil)
var _ SyncChecker = (*ValidatorService)(nil)

func TestStop_CancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	vs := &ValidatorService{
		ctx:    ctx,
		cancel: cancel,
	}

	assert.NoError(t, vs.Stop())

	select {
	case <-time.After(1 * time.Second):
		t.Error("Context not canceled within 1s")
	case <-vs.ctx.Done():
	}
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use canceled context so that the run function exits immediately..
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validatorService := &ValidatorService{
		ctx:      ctx,
		cancel:   cancel,
		endpoint: "merkle tries",
		withCert: "alice.crt",
	}
	validatorService.Start()
	require.NoError(t, validatorService.Stop(), "Could not stop service")
	require.LogsContain(t, hook, "Stopping service")
}

func TestLifecycle_Insecure(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use canceled context so that the run function exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validatorService := &ValidatorService{
		ctx:      ctx,
		cancel:   cancel,
		endpoint: "merkle tries",
	}
	validatorService.Start()
	require.LogsContain(t, hook, "You are using an insecure gRPC connection")
	require.NoError(t, validatorService.Stop(), "Could not stop service")
	require.LogsContain(t, hook, "Stopping service")
}

func TestStatus_NoConnectionError(t *testing.T) {
	validatorService := &ValidatorService{}
	assert.ErrorContains(t, "no connection", validatorService.Status())
}

func TestStart_GrpcHeaders(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use canceled context so that the run function exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for input, output := range map[string][]string{
		"should-break": []string{},
		"key=value":    []string{"key", "value"},
		"":             []string{},
		",":            []string{},
		"key=value,Authorization=Q=": []string{
			"key", "value", "Authorization", "Q=",
		},
		"Authorization=this is a valid value": []string{
			"Authorization", "this is a valid value",
		},
	} {
		validatorService := &ValidatorService{
			ctx:         ctx,
			cancel:      cancel,
			endpoint:    "merkle tries",
			grpcHeaders: strings.Split(input, ","),
		}
		validatorService.Start()
		md, _ := metadata.FromOutgoingContext(validatorService.ctx)
		if input == "should-break" {
			require.LogsContain(t, hook, "Incorrect gRPC header flag format. Skipping should-break")
		} else if len(output) == 0 {
			require.DeepEqual(t, md, metadata.MD(nil))
		} else {
			require.DeepEqual(t, md, metadata.Pairs(output...))
		}
	}
}

func TestHandleAccountChanges(t *testing.T) {
	logHook := logTest.NewGlobal()
	ctx := context.Background()
	password := "Passw03rdz293**%#2"

	// Prepare keys
	originalPrivKey, err := bls.RandKey()
	require.NoError(t, err)
	originalPubKeyBytes := originalPrivKey.PublicKey().Marshal()
	newPrivKey, err := bls.RandKey()
	require.NoError(t, err)
	newPubKeyBytes := newPrivKey.PublicKey().Marshal()

	// Prepare database
	db := dbTest.SetupDB(t, [][48]byte{bytesutil.ToBytes48(originalPubKeyBytes)})

	// Prepare validator client mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := mock.NewMockBeaconNodeValidatorClient(ctrl)
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	validatorClient.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{originalPubKeyBytes, newPubKeyBytes},
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ValidatorActivationResponse{
			Statuses: []*ethpb.ValidatorActivationResponse_Status{
				{
					PublicKey: originalPubKeyBytes,
					Status:    &ethpb.ValidatorStatusResponse{Status: ethpb.ValidatorStatus_ACTIVE},
				},
				{
					PublicKey: newPubKeyBytes,
					Status:    &ethpb.ValidatorStatusResponse{Status: ethpb.ValidatorStatus_UNKNOWN_STATUS},
				},
			},
		},
		nil,
	)

	// Prepare keymanager
	walletDir := t.TempDir()
	wallet, err := accounts.CreateWalletWithKeymanager(ctx, &accounts.CreateWalletConfig{
		WalletCfg: &wallet2.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Imported,
			WalletPassword: password,
		},
	})
	require.NoError(t, err)
	km, err := imported.NewKeymanager(ctx, &imported.SetupConfig{
		Wallet: wallet,
	})
	require.NoError(t, err)
	require.NoError(t, km.ImportKeystores(ctx, []*keymanager.Keystore{createKeystore(t, password, originalPrivKey)}, password))

	// Prepare validator service
	validatorService := &ValidatorService{
		ctx: ctx,
		db:  db,
		validator: &validator{
			keyManager:      km,
			validatorClient: validatorClient,
		},
		keyManager: km,
	}

	// Run the test: subscribe to account changes and simulate such changes by importing a new keystore
	go validatorService.handleAccountChanges(ctx)
	require.NoError(t, km.ImportKeystores(ctx, []*keymanager.Keystore{createKeystore(t, password, newPrivKey)}, password))
	time.Sleep(time.Second * 5) // Allow code subscribed to account changes to run

	// Assert
	pubKeys, err := db.ProposedPublicKeys(ctx)
	originalFound := false
	newFound := false
	for _, key := range pubKeys {
		if key == bytesutil.ToBytes48(originalPubKeyBytes) {
			originalFound = true
		} else if key == bytesutil.ToBytes48(newPubKeyBytes) {
			newFound = true
		}
	}
	assert.Equal(t, true, originalFound, "original key was removed from the database")
	assert.Equal(t, true, newFound, "new key was not added to the database")
	assert.LogsContain(t, logHook, fmt.Sprintf("%#x", bytesutil.Trunc(originalPubKeyBytes)))
	assert.LogsContain(t, logHook, fmt.Sprintf("%#x", bytesutil.Trunc(newPubKeyBytes)))
	assert.LogsContain(t, logHook, "Waiting for deposit to be observed by beacon node")
}

func createKeystore(t testing.TB, password string, privKey bls.SecretKey) *keymanager.Keystore {
	encryptor := keystorev4.New()
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), password)
	require.NoError(t, err)
	return &keymanager.Keystore{
		Crypto:  cryptoFields,
		Pubkey:  fmt.Sprintf("%x", privKey.PublicKey().Marshal()),
		ID:      id.String(),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
}

package polling

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = shared.Service(&ValidatorService{})
var validatorKey *keystore.Key
var validatorPubKey [48]byte
var keyMap map[[48]byte]*keystore.Key
var keyMapThreeValidators map[[48]byte]*keystore.Key
var testKeyManager keymanager.KeyManager
var testKeyManagerThreeValidators keymanager.KeyManager

func keySetup() {
	keyMap = make(map[[48]byte]*keystore.Key)
	keyMapThreeValidators = make(map[[48]byte]*keystore.Key)

	var err error
	validatorKey, err = keystore.NewKey()
	if err != nil {
		log.WithError(err).Debug("Cannot create key")
	}
	copy(validatorPubKey[:], validatorKey.PublicKey.Marshal())
	keyMap[validatorPubKey] = validatorKey

	sks := make([]bls.SecretKey, 1)
	sks[0] = validatorKey.SecretKey
	testKeyManager = keymanager.NewDirect(sks)

	sks = make([]bls.SecretKey, 3)
	for i := 0; i < 3; i++ {
		vKey, err := keystore.NewKey()
		if err != nil {
			log.WithError(err).Debug("Cannot create key")
		}
		var pubKey [48]byte
		copy(pubKey[:], vKey.PublicKey.Marshal())
		keyMapThreeValidators[pubKey] = vKey
		sks[i] = vKey.SecretKey
	}
	testKeyManagerThreeValidators = keymanager.NewDirect(sks)
}

func TestMain(m *testing.M) {
	dir := testutil.TempDir() + "/keystore1"
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.WithError(err).Debug("Cannot remove keystore folder")
		}
	}()
	if err := accounts.NewValidatorAccount(dir, "1234"); err != nil {
		log.WithError(err).Debug("Cannot create validator account")
	}
	keySetup()
	os.Exit(m.Run())
}

func TestStop_CancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	vs := &ValidatorService{
		ctx:    ctx,
		cancel: cancel,
	}

	if err := vs.Stop(); err != nil {
		t.Error(err)
	}

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
		ctx:        ctx,
		cancel:     cancel,
		endpoint:   "merkle tries",
		withCert:   "alice.crt",
		keyManager: keymanager.NewDirect(nil),
	}
	validatorService.Start()
	if err := validatorService.Stop(); err != nil {
		t.Fatalf("Could not stop service: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestLifecycle_Insecure(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use canceled context so that the run function exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validatorService := &ValidatorService{
		ctx:        ctx,
		cancel:     cancel,
		endpoint:   "merkle tries",
		keyManager: keymanager.NewDirect(nil),
	}
	validatorService.Start()
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")
	if err := validatorService.Stop(); err != nil {
		t.Fatalf("Could not stop service: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestStatus_NoConnectionError(t *testing.T) {
	validatorService := &ValidatorService{}
	if err := validatorService.Status(); !strings.Contains(err.Error(), "no connection") {
		t.Errorf("Expected status check to fail if no connection is found, received: %v", err)
	}
}

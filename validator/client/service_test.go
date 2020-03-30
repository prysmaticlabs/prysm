package client

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = shared.Service(&ValidatorService{})
var validatorPubKey *bls.PublicKey
var secKeyMap map[[48]byte]*bls.SecretKey
var pubKeyMap map[[48]byte]*bls.PublicKey
var secKeyMapThreeValidators map[[48]byte]*bls.SecretKey
var pubKeyMapThreeValidators map[[48]byte]*bls.PublicKey
var testKeyManager keymanager.KeyManager
var testKeyManagerThreeValidators keymanager.KeyManager

func keySetup() {
	pubKeyMap = make(map[[48]byte]*bls.PublicKey)
	secKeyMap = make(map[[48]byte]*bls.SecretKey)
	pubKeyMapThreeValidators = make(map[[48]byte]*bls.PublicKey)
	secKeyMapThreeValidators = make(map[[48]byte]*bls.SecretKey)

	sks := make([]*bls.SecretKey, 1)
	sks[0] = bls.RandKey()
	testKeyManager = keymanager.NewDirect(sks)
	validatorPubKey = sks[0].PublicKey()

	sks = make([]*bls.SecretKey, 3)
	for i := 0; i < 3; i++ {
		secKey := bls.RandKey()
		var pubKey [48]byte
		copy(pubKey[:], secKey.PublicKey().Marshal())
		secKeyMapThreeValidators[pubKey] = secKey
		pubKeyMapThreeValidators[pubKey] = secKey.PublicKey()
		sks[i] = secKey
	}
	testKeyManagerThreeValidators = keymanager.NewDirect(sks)
}

func TestMain(m *testing.M) {
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

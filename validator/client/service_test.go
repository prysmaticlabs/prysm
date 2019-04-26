package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = shared.Service(&ValidatorService{})
var validatorKey *keystore.Key
var keyMap map[string]*keystore.Key
var keyMapThreeValidators map[string]*keystore.Key

func keySetup() {
	keyMap = make(map[string]*keystore.Key)
	keyMapThreeValidators = make(map[string]*keystore.Key)

	validatorKey, _ = keystore.NewKey(rand.Reader)
	keyMap[hex.EncodeToString(validatorKey.PublicKey.Marshal())] = validatorKey

	for i := 0; i < 3; i++ {
		vKey, _ := keystore.NewKey(rand.Reader)
		keyMapThreeValidators[hex.EncodeToString(vKey.PublicKey.Marshal())] = vKey
	}
}

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
}

func TestMain(m *testing.M) {
	dir := testutil.TempDir() + "/keystore1"
	defer os.RemoveAll(dir)
	accounts.NewValidatorAccount(dir, "1234")
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
		t.Error("ctx not cancelled within 1s")
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
		keys:     keyMap,
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
		ctx:      ctx,
		cancel:   cancel,
		endpoint: "merkle tries",
		keys:     keyMap,
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

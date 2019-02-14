package client

import (
	"context"
	"crypto/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/keystore"

	"github.com/prysmaticlabs/prysm/validator/accounts"

	"github.com/prysmaticlabs/prysm/shared/testutil"

	"github.com/prysmaticlabs/prysm/shared"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = shared.Service(&ValidatorService{})
var validatorKey *keystore.Key

func TestMain(m *testing.M) {
	dir := testutil.TempDir() + "/keystore1"
	defer os.RemoveAll(dir)
	accounts.NewValidatorAccount(dir, "1234")
	validatorKey, _ = keystore.NewKey(rand.Reader)
	os.Exit(m.Run())
}

func TestStop_cancelsContext(t *testing.T) {
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
	// Use cancelled context so that the run function exits immediately..
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validatorService := &ValidatorService{
		ctx:      ctx,
		cancel:   cancel,
		endpoint: "merkle tries",
		withCert: "alice.crt",
		key:      validatorKey,
	}
	validatorService.Start()
	if err := validatorService.Stop(); err != nil {
		t.Fatalf("Could not stop service: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestLifecycle_WithInsecure(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use cancelled context so that the run function exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validatorService := &ValidatorService{
		ctx:      ctx,
		cancel:   cancel,
		endpoint: "merkle tries",
		key:      validatorKey,
	}
	validatorService.Start()
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")
	if err := validatorService.Stop(); err != nil {
		t.Fatalf("Could not stop service: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestStatus(t *testing.T) {
	validatorService := &ValidatorService{}
	if err := validatorService.Status(); !strings.Contains(err.Error(), "no connection") {
		t.Errorf("Expected status check to fail if no connection is found, received: %v", err)
	}
}

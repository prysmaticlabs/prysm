package client

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/testutil"

	"github.com/prysmaticlabs/prysm/shared"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = shared.Service(&ValidatorService{})

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
	validatorService, err := NewValidatorService(
		ctx,
		&Config{
			Endpoint: "merkle tries",
			CertFlag: "alice.crt",
		},
	)
	if err != nil {
		t.Fatal(err)
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
	validatorService, err := NewValidatorService(
		ctx,
		&Config{
			Endpoint: "merkle tries",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	validatorService.Start()
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")
	if err := validatorService.Stop(); err != nil {
		t.Fatalf("Could not stop service: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestStatus(t *testing.T) {
	validatorService, err := NewValidatorService(
		context.Background(),
		&Config{
			Endpoint: "merkle tries",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatorService.Status(); !strings.Contains(err.Error(), "no connection") {
		t.Errorf("Expected status check to fail if no connection is found, received: %v", err)
	}
}

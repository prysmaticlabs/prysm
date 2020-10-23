package client

import (
	"context"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ shared.Service = (*ValidatorService)(nil)

func TestMain(m *testing.M) {
	dir := testutil.TempDir() + "/keystore1"
	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			log.WithError(err).Debug("Cannot remove keystore folder")
		}
	}
	defer cleanup()
	code := m.Run()
	// os.Exit will prevent defer from being called
	cleanup()
	os.Exit(code)
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use canceled context so that the run function exits immediately..
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validatorService := &ValidatorService{
		endpoint: "merkle tries",
		withCert: "alice.crt",
	}
	validatorService.Start(ctx)
	require.NoError(t, validatorService.Stop(ctx), "Could not stop service")
	require.LogsContain(t, hook, "Stopping service")
}

func TestLifecycle_Insecure(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use canceled context so that the run function exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validatorService := &ValidatorService{
		endpoint: "merkle tries",
	}
	validatorService.Start(ctx)
	require.LogsContain(t, hook, "You are using an insecure gRPC connection")
	require.NoError(t, validatorService.Stop(ctx), "Could not stop service")
	require.LogsContain(t, hook, "Stopping service")
}

func TestStatus_NoConnectionError(t *testing.T) {
	validatorService := &ValidatorService{}
	assert.ErrorContains(t, "no connection", validatorService.Status())
}

package rpc

import (
	"context"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLifecycle_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewService(&Config{
		Port:     "7348",
		CertFlag: "alice.crt",
		KeyFlag:  "alice.key",
	})

	ctx := context.Background()
	rpcService.Start(ctx)

	require.LogsContain(t, hook, "listening on port")
	require.NoError(t, rpcService.Stop(ctx))
}

func TestStatus_CredentialError(t *testing.T) {
	credentialErr := errors.New("credentialError")
	s := &Service{credentialError: credentialErr}

	assert.ErrorContains(t, s.credentialError.Error(), s.Status())
}

func TestRPC_InsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewService(&Config{
		Port: "7777",
	})

	ctx := context.Background()
	rpcService.Start(ctx)

	require.LogsContain(t, hook, "listening on port")
	require.NoError(t, rpcService.Stop(ctx))
}

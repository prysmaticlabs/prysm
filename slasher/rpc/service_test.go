package rpc

import (
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLifecycle_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService, serviceCtx := NewService(&Config{
		Port:     "7348",
		CertFlag: "alice.crt",
		KeyFlag:  "alice.key",
	})

	rpcService.Start(serviceCtx.Ctx)

	require.LogsContain(t, hook, "listening on port")
	require.NoError(t, rpcService.Stop(serviceCtx.Ctx))
}

func TestStatus_CredentialError(t *testing.T) {
	credentialErr := errors.New("credentialError")
	s := &Service{credentialError: credentialErr}

	assert.ErrorContains(t, s.credentialError.Error(), s.Status())
}

func TestRPC_InsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService, serviceCtx := NewService(&Config{
		Port: "7777",
	})

	rpcService.Start(serviceCtx.Ctx)

	require.LogsContain(t, hook, "listening on port")
	require.NoError(t, rpcService.Stop(serviceCtx.Ctx))
}

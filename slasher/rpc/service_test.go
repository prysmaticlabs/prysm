package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestLifecycle_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewService(context.Background(), &Config{
		Port:     "7348",
		CertFlag: "alice.crt",
		KeyFlag:  "alice.key",
	})

	rpcService.Start()

	require.LogsContain(t, hook, "listening on port")

	if err := rpcService.Stop(); err != nil {
		t.Error(err)
	}
}

func TestStatus_CredentialError(t *testing.T) {
	credentialErr := errors.New("credentialError")
	s := &Service{credentialError: credentialErr}

	if err := s.Status(); err != s.credentialError {
		t.Errorf("Wanted: %v, got: %v", s.credentialError, s.Status())
	}
}

func TestRPC_InsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	rpcService := NewService(context.Background(), &Config{
		Port: "7777",
	})

	rpcService.Start()

	require.LogsContain(t, hook, fmt.Sprint("listening on port"))

	if err := rpcService.Stop(); err != nil {
		t.Error(err)
	}
}
